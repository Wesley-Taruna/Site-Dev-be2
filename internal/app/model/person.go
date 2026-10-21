package model

import (
	"encoding/json"
	"time"
)

type Person struct {
	ID        int64     `orm:"column(id)" json:"id"`
	Name      string    `orm:"column(name)" json:"name"`
	City      string    `orm:"column(city)" json:"city"`
	Country   string    `orm:"column(country)" json:"country"`
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
	UpdatedAt time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updated_at"`
}

func init() {
	// orm.DB().AutoMigrate(&Person{})
}

func (m *Person) MarshalJSON() ([]byte, error) {
	type Alias Person

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
