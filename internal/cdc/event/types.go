package event

//nolint:tagliatelle // Debezium 标准字段使用 "__xx" 格式。
type CDCEvent struct {
	Op    string `json:"__op"`
	DB    string `json:"__db"`
	Table string `json:"__table"`
	TSMs  int64  `json:"__ts_ms"`
}

func (e CDCEvent) IsCreate() bool {
	return e.Op == "c"
}

func (e CDCEvent) IsUpdate() bool {
	return e.Op == "u"
}

func (e CDCEvent) IsDelete() bool {
	return e.Op == "d"
}
