package model

type AddressModel struct {
	Pref      string
	Street    []int
	IntValues []int
	Date      *Date1
}

type Date1 struct {
	Year string
}
