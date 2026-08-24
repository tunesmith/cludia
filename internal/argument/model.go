package argument

type Role string

const (
	RolePremise      Role = "premise"
	RoleLemma        Role = "lemma"
	RoleConclusion   Role = "conclusion"
	RoleCounterpoint Role = "counterpoint"
)

type Kind string

const (
	KindFact  Kind = "fact"
	KindValue Kind = "value"
)

type Truth string

const (
	TruthTrue    Truth = "T"
	TruthFalse   Truth = "F"
	TruthUnknown Truth = "U"
)

type Connector string

const (
	ConnectorAND Connector = "AND"
	ConnectorOR  Connector = "OR"
)

type DefeatScope string

const (
	DefeatPremise      DefeatScope = "premise"
	DefeatInference    DefeatScope = "inference"
	DefeatCounterpoint DefeatScope = "counterpoint"
)

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Document struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Metadata       []Metadata      `json:"metadata"`
	Statements     []Statement     `json:"statements"`
	Junctors       []Junctor       `json:"junctors"`
	DirectSupports []DirectSupport `json:"direct_supports"`
	Defeats        []Defeat        `json:"defeats"`
}

type Statement struct {
	ID    string `json:"id"`
	Slug  string `json:"slug,omitempty"`
	Role  Role   `json:"role"`
	Kind  Kind   `json:"kind"`
	Truth Truth  `json:"truth"`
	Text  string `json:"text"`
}

type Junctor struct {
	ID        string    `json:"id"`
	Connector Connector `json:"connector"`
	Sources   []string  `json:"sources"`
	Target    string    `json:"target"`
	Order     int       `json:"-"`
}

type DirectSupport struct {
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Connector Connector `json:"connector"`
	Order     int       `json:"-"`
}

type Defeat struct {
	From      string      `json:"from"`
	Scope     DefeatScope `json:"scope"`
	To        string      `json:"to,omitempty"`
	JunctorID string      `json:"junctor_id,omitempty"`
	AtTarget  string      `json:"at_target,omitempty"`
}

func (d *Document) MetadataValue(key string) (string, bool) {
	for i := len(d.Metadata) - 1; i >= 0; i-- {
		if d.Metadata[i].Key == key {
			return d.Metadata[i].Value, true
		}
	}
	return "", false
}

func (d *Document) Statement(idOrSlug string) (*Statement, bool) {
	for i := range d.Statements {
		if d.Statements[i].ID == idOrSlug || d.Statements[i].Slug == idOrSlug {
			return &d.Statements[i], true
		}
	}
	return nil, false
}

func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}
	next := *d
	next.Metadata = append([]Metadata(nil), d.Metadata...)
	next.Statements = append([]Statement(nil), d.Statements...)
	next.Junctors = append([]Junctor(nil), d.Junctors...)
	for i := range next.Junctors {
		next.Junctors[i].Sources = append([]string(nil), d.Junctors[i].Sources...)
	}
	next.DirectSupports = append([]DirectSupport(nil), d.DirectSupports...)
	next.Defeats = append([]Defeat(nil), d.Defeats...)
	return &next
}
