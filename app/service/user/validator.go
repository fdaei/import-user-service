package user

import (
	"errors"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	errmsg "rankr/pkg/err_msg"
	"rankr/pkg/validator"
	types "rankr/type"
)

type validatorImpl struct{}

func NewValidator() Validator {
	return validatorImpl{}
}

func (validatorImpl) ValidateImportUser(u ImportUser) error {
	err := validation.ValidateStruct(&u,
		validation.Field(&u.ID, validation.Required),
		validation.Field(&u.Name, validation.Required),
	)
	if err != nil {
		return validator.NewError(err, validator.Flat, "invalid user payload")
	}

	for i, addr := range u.Addresses {
		if addr.Street == "" {
			return validator.NewError(
				fmt.Errorf("address line is required at index %d", i),
				validator.Flat,
				errmsg.ErrValidationFailed.Error(),
			)
		}
	}

	return nil
}

func (validatorImpl) ValidateUserID(id types.ID) error {
	if id == 0 {
		return validator.NewError(errors.New("id must be greater than zero"), validator.Flat, "invalid user id")
	}
	return nil
}
