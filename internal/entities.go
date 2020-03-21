package internal

type EI struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
}

type Param struct {
	IdParamOwner int    `json:"-"`
	Id           int    `json:"id"`
	Name         string `json:"name"`
	ValType      string `json:"val_type"`
	EI           *EI    `json:"ei"`
}

type ParamAndValues struct {
	Param *Param      `json:"param"`
	Value interface{} `json:"value"`
}

type Class struct {
	Id       int      `json:"id"`
	Name     string   `json:"name"`
	Children []*Class `json:"children"`
	Ei       *EI      `json:"ei"`
	Params   []*Param `json:"params"`
}

type Product struct {
	Id          int               `json:"id"`
	Name        string            `json:"name"`
	ParentClass *Class            `json:"parent_class"`
	Params      []*ParamAndValues `json:"params"`
}
