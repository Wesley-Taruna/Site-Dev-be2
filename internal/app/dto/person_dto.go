package dto

import "time"

type PersonRequestGet struct {
	ID      int64  `json:"id" valid:"required"`
	Name    string `json:"name" valid:"required"`
	City    string `json:"city" valid:"required"`
	Country string `json:"country" valid:"required"`
}

type PersonResponseGet struct {
	ID        int64     `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	City      string    `json:"city,omitempty"`
	Country   string    `json:"country,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type PersonRequestCreate struct {
	Name    string `json:"name" valid:"required"`
	City    string `json:"city" valid:"required"`
	Country string `json:"country" valid:"required"`
}

type PersonResponseCreate struct {
	ID        int64     `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	City      string    `json:"city,omitempty"`
	Country   string    `json:"country,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type PersonRequestUpdate struct {
	ID      int64
	Name    string `json:"name" valid:"required"`
	City    string `json:"city" valid:"required"`
	Country string `json:"country" valid:"required"`
}

type PersonResponseUpdate struct {
	ID        int64     `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	City      string    `json:"city,omitempty"`
	Country   string    `json:"country,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type PersonRequestDelete struct {
	ID   int64
	Note string `json:"note" valid:"required"`
}

type PersonResponseDelete struct {
	Note string `json:"note,omitempty"`
}
