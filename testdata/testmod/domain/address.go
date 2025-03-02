package domain

type Address struct {
	Pref         string
	Street       string
	StringValues []string
	Date         Date1
}

type Date1 struct {
	Year int
}
