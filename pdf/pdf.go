package pdf

import (
	"errors"
	"fmt"
	"github.com/1lann/go-pdfreader"
	"github.com/1lann/go-pdfreader/cmapi"
	"github.com/1lann/go-pdfreader/fancy"
	"github.com/1lann/go-pdfreader/ps"
	"github.com/1lann/go-tax/dispenser"
	"github.com/1lann/go-tax/statement"
	"runtime/debug"
	"strings"
	"time"
)

type headerType int

const (
	headerFranked headerType = iota
	headerUnfranked
	headerFrankingCredit
	headerWithholdingTax
	headerSharesAllotted
	headerCostOfSharesAllotted
	headerTotalShares
	headerTotalPayment
	headerOther
)

func (h headerType) String() string {
	switch h {
	case headerFranked:
		return "Franked"
	case headerUnfranked:
		return "Unfranked"
	case headerFrankingCredit:
		return "Franking Credit"
	case headerWithholdingTax:
		return "Withholding Tax"
	case headerSharesAllotted:
		return "Shares Allotted"
	case headerCostOfSharesAllotted:
		return "Cost Of Shares Allotted"
	case headerTotalShares:
		return "Total Shares"
	case headerTotalPayment:
		return "Total Payment"
	case headerOther:
		return "Other"
	default:
		return "Unknown"
	}
}

type textTracker struct {
	text  []string
	pdf   *pdfread.PdfReaderT
	page  int
	cmaps map[string]*cmapi.CharMapperT
	fonts pdfread.DictionaryT
	font  string
	stack [][]byte
}

func (t *textTracker) cmap(font string) (r *cmapi.CharMapperT) {
	var ok bool
	if r, ok = t.cmaps[font]; ok {
		return
	}
	r = cmapi.Read(nil)
	if t.fonts == nil {
		t.fonts = t.pdf.PageFonts(t.pdf.Pages()[t.page])
		if t.fonts == nil {
			return
		}
	}
	if dr, ok := t.fonts[font]; ok {
		d := t.pdf.Dic(dr)
		if tu, ok := d["/ToUnicode"]; ok {
			_, cm := t.pdf.DecodedStream(tu)
			r = cmapi.Read(fancy.SliceReader(cm))
			t.cmaps[font] = r
		}
	}
	return
}

func (t *textTracker) write(a []byte) {
	tx := t.pdf.ForcedArray(a)
	for k := range tx {
		if tx[k][0] == '(' || tx[k][0] == '<' {
			str := string(cmapi.Decode(ps.String(tx[k]), t.cmap(t.font)))
			if str == " " && len(t.text) > 0 {
				t.text[len(t.text)-1] += " "
			} else {
				t.text = append(t.text, str)
			}
		}
	}
}

func (t *textTracker) push(a []byte) {
	t.stack = append(t.stack, a)
}

func (t *textTracker) drop(num int) [][]byte {
	result := t.stack[len(t.stack)-num:]
	t.stack = t.stack[:len(t.stack)-num]
	return result
}

func (t *textTracker) process(data []byte) {
	rd := fancy.SliceReader(data)
	for {
		token, _ := ps.Token(rd)
		if len(token) == 0 {
			break
		}

		switch string(token) {
		case "B", "B*", "F", "S", "b", "b*", "f", "f*", "h", "n", "s", "BT",
			"ET", "T*", "EMC":
		case "G", "J", "M", "g", "gs", "i", "j", "w", "TL", "Tc",
			"Tr", "Ts", "Tw", "Tz", "BMC", "MP":
			t.drop(1)
		case "l", "m", "TD", "Td", "BDC", "DP":
			t.drop(2)
		case "RG", "rg":
			t.drop(3)
		case "re", "v", "y", "K", "k":
			t.drop(4)
		case "c", "cm", "Tm":
			t.drop(6)
		case "Tf":
			t.font = string(t.drop(2)[0])
		case "'", "TJ", "Tj":
			t.write(t.drop(1)[0])
		case "\"":
			t.write(t.drop(3)[2])
		default:
			t.push(token)
		}
	}
}

func Process(filename string, holders []string) (s statement.Statement, err error) {
	defer func() {
		if r := recover(); r != nil {
			rErr, ok := r.(error)
			if ok {
				err = errors.New("pdf: failed to process file: " +
					rErr.Error() + ": " + string(debug.Stack()))
			} else {
				err = errors.New("pdf: failed to process file: " +
					string(debug.Stack()))
			}

		}
	}()

	pd := pdfread.Load(filename)
	_, data := pd.DecodedStream(pd.ForcedArray(pd.Dic(
		pd.Pages()[0])["/Contents"])[0])

	ttracker := &textTracker{
		pdf:   pd,
		page:  0,
		cmaps: make(map[string]*cmapi.CharMapperT),
	}

	ttracker.process(data)
	return processText(ttracker.text, holders)
}

var numeralHeaders = map[headerType][]string{
	headerFranked:              {"franked amount"},
	headerUnfranked:            {"unfranked"},
	headerWithholdingTax:       {"withholding tax", "less withholding tax"},
	headerSharesAllotted:       {"number of shares allotted"},
	headerCostOfSharesAllotted: {"cost of shares allotted"},
	headerTotalShares:          {"total shares"},
	headerTotalPayment:         {"total payment", "total amount"},
	headerFrankingCredit:       {"franking credit"},
	headerOther: {"dividend rate", "participating shares",
		"participating holding", "net amount",
		"dividend reinvestment plan amount",
		"cash balance brought forward", "amount available from this payment",
		"total amount available for reinvestment",
		"cash balance carried forward"},
}

func numeralHeader(str string, state statement.Statement) (headerType, bool) {
	str = strings.ToLower(str)
	for headType, header := range numeralHeaders {
		for _, headerText := range header {
			if len(str) >= len(headerText) && str[:len(headerText)] == headerText {
				switch headType {
				case headerFranked:
					if state.FrankedAmount.HasValue {
						continue
					}
				case headerUnfranked:
					if state.UnfrankedAmount.HasValue {
						continue
					}
				case headerWithholdingTax:
					if state.WithholdingTax.HasValue {
						continue
					}
				case headerSharesAllotted:
					if state.SharesAllotted != 0 {
						continue
					}
				case headerCostOfSharesAllotted:
					if state.CostOfSharesAllotted.HasValue {
						continue
					}
				case headerTotalShares:
					if state.TotalShares != 0 {
						continue
					}
				case headerTotalPayment:
					if state.TotalPayment.HasValue {
						continue
					}
				case headerFrankingCredit:
					if state.FrankingCredit.HasValue {
						continue
					}
				}

				return headType, true
			}
		}
	}

	return 0, false
}

func processText(str []string, holders []string) (statement.Statement, error) {
	d := dispenser.NewDispenser(str)

	var state statement.Statement
	var numeralHeadTracker []headerType

	for d.NextSentence() {
		sentence := d.DumpSentence()

		headType, foundNumeralHeader := numeralHeader(sentence, state)
		if foundNumeralHeader {
			numeralHeadTracker = append(numeralHeadTracker, headType)
			continue
		} else {
			d.StartOfSentence()
			newSentence := strings.Join(d.DumpNSentences(5), " ")

			if len(state.AccountHolders) == 0 {
				for _, holder := range holders {
					if strings.Contains(strings.ToLower(newSentence),
						strings.ToLower(holder)) {
						state.AccountHolders = append(state.AccountHolders, holder)
					}
				}
			}

			headType, foundNumeralHeader = numeralHeader(newSentence, state)
			if foundNumeralHeader {
				numeralHeadTracker = append(numeralHeadTracker, headType)
				continue
			}
		}

		findOtherData(sentence, d, &state)

		if len(numeralHeadTracker) > 0 {
			numD := dispenser.NewDispenserFromSentence(sentence)
			numD.NextSentence()
			found := numD.JumpNextNumeral()
			if found && numD.Position() < 5 {
				switch numeralHeadTracker[0] {
				case headerFranked:
					state.FrankedAmount = statement.NewDollar(numD.Numeral())
				case headerUnfranked:
					state.UnfrankedAmount = statement.NewDollar(numD.Numeral())
				case headerWithholdingTax:
					state.WithholdingTax = statement.NewDollar(numD.Numeral())
				case headerSharesAllotted:
					state.SharesAllotted = int(numD.Numeral())
				case headerCostOfSharesAllotted:
					state.CostOfSharesAllotted = statement.NewDollar(numD.Numeral())
				case headerTotalShares:
					state.TotalShares = int(numD.Numeral())
				case headerTotalPayment:
					state.TotalPayment = statement.NewDollar(numD.Numeral())
				case headerFrankingCredit:
					state.FrankingCredit = statement.NewDollar(numD.Numeral())
				}

				numeralHeadTracker = numeralHeadTracker[1:]
			}
		}
	}

	return state, nil
}

const dateLayout = "02 January 2006"

func findOtherData(sentence string, d *dispenser.Dispenser,
	state *statement.Statement) {
	if len(sentence) > 3 && sentence[:3] == "ABN" {
		d.LastSentence()
		sent := d.DumpSentence()
		state.Entity = sent
		d.NextSentence()
	}

	if len(sentence) > 10 &&
		strings.ToLower(sentence[:10]) == "asx code: " {
		state.ASXCode = strings.ToUpper(sentence[10:])
	}

	if strings.Contains(strings.ToLower(sentence), "payment date") ||
		strings.Contains(strings.ToLower(sentence), "holder reference number") {
		d.NextSentence()
		sent := d.DumpSentence()
		t, err := time.Parse(dateLayout, sent)
		if err != nil {
			fmt.Println("failed to parse time:", err)
			fmt.Println("from:", sent)
		} else {
			state.PaymentDate = t
		}
		d.LastSentence()
	}
}
