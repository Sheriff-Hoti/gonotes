package main

import "time"

// Go mirror of the TypeScript card-type API
// (frontend/src/lib/jev/types.ts, frontend/src/lib/parse/*,
// frontend/src/lib/savedItems.ts). JSON tags match the TS field names so
// values cross the Wails bindings boundary unchanged. Date maps to time.Time
// (nullable dates are *time.Time).

// ---------------------------------------------------------------------------
// Intent keys
// ---------------------------------------------------------------------------

// IntentKey is every card type plus "none" (no confident match).
type IntentKey string

const (
	IntentEvent     IntentKey = "event"
	IntentReminder  IntentKey = "reminder"
	IntentTodo      IntentKey = "todo"
	IntentTimer     IntentKey = "timer"
	IntentHabit     IntentKey = "habit"
	IntentColor     IntentKey = "color"
	IntentSplit     IntentKey = "split"
	IntentExpense   IntentKey = "expense"
	IntentConvert   IntentKey = "convert"
	IntentCalc      IntentKey = "calc"
	IntentTravel    IntentKey = "travel"
	IntentPoll      IntentKey = "poll"
	IntentContact   IntentKey = "contact"
	IntentLink      IntentKey = "link"
	IntentCountdown IntentKey = "countdown"
	IntentTimezone  IntentKey = "timezone"
	IntentRandom    IntentKey = "random"
	IntentGoal      IntentKey = "goal"
	IntentNote      IntentKey = "note"
	IntentNone      IntentKey = "none"
)

// CardIntent is any intent that renders a card (everything but "none").
type CardIntent = IntentKey

// ---------------------------------------------------------------------------
// Signal vocabularies
// ---------------------------------------------------------------------------

type Tone string

const (
	ToneNeutral    Tone = "neutral"
	TonePositive   Tone = "positive"
	ToneExcited    Tone = "excited"
	ToneStressed   Tone = "stressed"
	ToneReflective Tone = "reflective"
)

type EventMode string

const (
	EventModeInPerson    EventMode = "in_person"
	EventModeVideoCall   EventMode = "video_call"
	EventModePhoneCall   EventMode = "phone_call"
	EventModeUnspecified EventMode = "unspecified"
)

type Transport string

const (
	TransportFlight      Transport = "flight"
	TransportTrain       Transport = "train"
	TransportBus         Transport = "bus"
	TransportCar         Transport = "car"
	TransportUnspecified Transport = "unspecified"
)

type TripType string

const (
	TripTypeWork        TripType = "work"
	TripTypeLeisure     TripType = "leisure"
	TripTypeUnspecified TripType = "unspecified"
)

type ExpenseCategory string

const (
	ExpenseFood          ExpenseCategory = "food"
	ExpenseTransport     ExpenseCategory = "transport"
	ExpenseShopping      ExpenseCategory = "shopping"
	ExpenseBills         ExpenseCategory = "bills"
	ExpenseEntertainment ExpenseCategory = "entertainment"
	ExpenseHealth        ExpenseCategory = "health"
	ExpenseOther         ExpenseCategory = "other"
)

type ColorMood string

const (
	ColorMoodWarm    ColorMood = "warm"
	ColorMoodCool    ColorMood = "cool"
	ColorMoodNeutral ColorMood = "neutral"
	ColorMoodVivid   ColorMood = "vivid"
	ColorMoodPastel  ColorMood = "pastel"
	ColorMoodDark    ColorMood = "dark"
)

type TimerKind string

const (
	TimerCountdown TimerKind = "countdown"
	TimerFocus     TimerKind = "focus"
	TimerBreak     TimerKind = "break"
	TimerStopwatch TimerKind = "stopwatch"
)

// Currency symbols as used by split/expense parsers. Default is rupee.
type Currency string

const (
	CurrencyINR Currency = "₹"
	CurrencyUSD Currency = "$"
	CurrencyEUR Currency = "€"
	CurrencyGBP Currency = "£"
)

// ColorSource says how a color was recognised.
type ColorSource string

const (
	ColorSourceHex   ColorSource = "hex"
	ColorSourceRGB   ColorSource = "rgb"
	ColorSourceNamed ColorSource = "named"
	ColorSourceMood  ColorSource = "mood"
)

// RandomKind discriminates RandomData below.
type RandomKind string

const (
	RandomDice   RandomKind = "dice"
	RandomCoin   RandomKind = "coin"
	RandomNumber RandomKind = "number"
	RandomPick   RandomKind = "pick"
)

// ---------------------------------------------------------------------------
// Classification result (IntentResult)
// ---------------------------------------------------------------------------

// Answer is one classified question: winning value, its confidence, and the
// full distribution over the alternatives.
type Answer[T ~string] struct {
	Value         T             `json:"value"`
	Confidence    float64       `json:"confidence"`
	Probabilities map[T]float64 `json:"probabilities"`
}

// Urgency is a score, not a category.
type Urgency struct {
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
}

// Signals are the 13 sub-answers accompanying every classification.
type Signals struct {
	IsQuestion         float64                 `json:"isQuestion"`
	Recurring          float64                 `json:"recurring"`
	Urgency            Urgency                 `json:"urgency"`
	Tone               Answer[Tone]            `json:"tone"`
	EventMode          Answer[EventMode]       `json:"eventMode"`
	Transport          Answer[Transport]       `json:"transport"`
	TripType           Answer[TripType]        `json:"tripType"`
	ExpenseCategory    Answer[ExpenseCategory] `json:"expenseCategory"`
	ColorMood          Answer[ColorMood]       `json:"colorMood"`
	TimerKind          Answer[TimerKind]       `json:"timerKind"`
	HasExplicitOptions float64                 `json:"hasExplicitOptions"`
	IsShoppingList     float64                 `json:"isShoppingList"`
}

// IntentResult is the full classification of one input text.
type IntentResult struct {
	Intent        Answer[IntentKey] `json:"intent"`
	Readiness     float64           `json:"readiness"`
	Signals       Signals           `json:"signals"`
	LatencyMs     int64             `json:"latencyMs"`
	QuestionCount int               `json:"questionCount"`
	Model         string            `json:"model"`
	Cached        bool              `json:"cached,omitempty"`
	Error         bool              `json:"error,omitempty"`
	Source        string            `json:"source,omitempty"` // "jev" | "mock"
}

// ---------------------------------------------------------------------------
// Parsed card data (ParsedMap). One struct per intent; nullables are pointers.
// ---------------------------------------------------------------------------

type EventData struct {
	Title    string     `json:"title"`
	Date     *time.Time `json:"date"`
	HasTime  bool       `json:"hasTime"`
	People   []string   `json:"people"`
	Link     *string    `json:"link"`
	Location *string    `json:"location"`
}

type ReminderData struct {
	Task    string     `json:"task"`
	When    *time.Time `json:"when"`
	HasTime bool       `json:"hasTime"`
}

type TodoData struct {
	Items []string `json:"items"`
	Verb  *string  `json:"verb"`
}

type TimerData struct {
	Seconds *int64 `json:"seconds"`
	Label   string `json:"label"`
}

type HabitData struct {
	Title   string  `json:"title"`
	Days    []int   `json:"days"`
	PerWeek *int    `json:"perWeek"`
	Label   *string `json:"label"`
}

type ColorData struct {
	Hex    *string      `json:"hex"`
	Name   *string      `json:"name"`
	Source *ColorSource `json:"source"`
}

type SplitData struct {
	Total    *float64 `json:"total"`
	People   *int     `json:"people"`
	Currency Currency `json:"currency"`
}

type ExpenseData struct {
	Amount   *float64 `json:"amount"`
	Item     string   `json:"item"`
	Currency Currency `json:"currency"`
}

type ConvertData struct {
	Value  *float64 `json:"value"`
	From   *string  `json:"from"`
	To     *string  `json:"to"`
	Result *float64 `json:"result"`
}

type CalcData struct {
	Expression string   `json:"expression"`
	Result     *float64 `json:"result"`
}

type TravelData struct {
	Destination *string    `json:"destination"`
	Origin      *string    `json:"origin"`
	Start       *time.Time `json:"start"`
	End         *time.Time `json:"end"`
}

type PollData struct {
	Title   string   `json:"title"`
	Options []string `json:"options"`
}

type ContactData struct {
	Name     string  `json:"name"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Initials string  `json:"initials"`
}

type LinkData struct {
	URL      *string `json:"url"`
	Domain   *string `json:"domain"`
	Monogram string  `json:"monogram"`
	Note     string  `json:"note"`
}

type CountdownData struct {
	Title string     `json:"title"`
	Date  *time.Time `json:"date"`
	Days  *int       `json:"days"`
}

// Zone is an abbreviation/city mapped to an IANA timezone.
type Zone struct {
	Label string `json:"label"`
	TZ    string `json:"tz"`
}

type TimezoneData struct {
	Instant *time.Time `json:"instant"`
	IsNow   bool       `json:"isNow"`
	From    Zone       `json:"from"`
	To      *Zone      `json:"to"`
}

// RandomData is a discriminated union on Kind; only the matching fields apply.
type RandomData struct {
	Kind    RandomKind `json:"kind"`
	Count   int        `json:"count,omitempty"`
	Sides   int        `json:"sides,omitempty"`
	Min     int        `json:"min,omitempty"`
	Max     int        `json:"max,omitempty"`
	Options []string   `json:"options,omitempty"`
}

type GoalData struct {
	Title   string   `json:"title"`
	Current float64  `json:"current"`
	Target  *float64 `json:"target"`
	Unit    *string  `json:"unit"`
}

type NoteData struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ---------------------------------------------------------------------------
// Persistence (savedItems)
// ---------------------------------------------------------------------------

// SavedItem is one completed card. CreatedAt is epoch milliseconds, matching
// the frontend store.
type SavedItem struct {
	ID        int64      `json:"id"`
	Intent    CardIntent `json:"intent"`
	Summary   string     `json:"summary"`
	Text      string     `json:"text"`
	CreatedAt int64      `json:"createdAt"`
}
