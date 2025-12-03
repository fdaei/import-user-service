package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	types "rankr/type"
)

var ErrNotFound = errors.New("user not found")

type Address struct {
	ID        types.ID  `json:"id,omitempty"`
	Street    string    `json:"street"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	ZipCode   string    `json:"zip_code"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type User struct {
	ID          types.ID  `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Addresses   []Address `json:"addresses"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type ImportUser struct {
	ID          NumericID       `json:"id"`
	Name        string          `json:"name"`
	Email       string          `json:"email"`
	PhoneNumber string          `json:"phone_number"`
	Addresses   []ImportAddress `json:"addresses"`
}

type ImportAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// convert json to user entity
func (u ImportUser) ToUser() User {
	addresses := make([]Address, 0, len(u.Addresses))
	for _, addr := range u.Addresses {
		addresses = append(addresses, Address{
			Street:  addr.Street,
			City:    addr.City,
			State:   addr.State,
			ZipCode: addr.ZipCode,
			Country: addr.Country,
		})
	}
	if addresses == nil {
		addresses = []Address{}
	}

	return User{
		ID:          u.ID.ToID(),
		Name:        u.Name,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		Addresses:   addresses,
	}
}

type NumericID uint64

func (n NumericID) ToID() types.ID {
	return types.ID(uint64(n))
}

func (n *NumericID) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("id is required")
	}

	if data[0] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return errors.New("id is required")
		}
		val, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("id must be numeric: %w", err)
		}
		*n = NumericID(val)
		return nil
	}

	var num uint64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*n = NumericID(num)
	return nil
}
