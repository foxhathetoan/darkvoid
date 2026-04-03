package service

import (
	"regexp"

	"github.com/jarviisha/darkvoid/internal/feature/user"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/validation"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)
	letterRegex   = regexp.MustCompile(`[a-zA-Z]`)
	numberRegex   = regexp.MustCompile(`[0-9]`)
)

const (
	minPasswordLength    = 8
	maxPasswordLength    = 72
	minDisplayNameLength = 1
	maxDisplayNameLength = 100
)

func validateCreateRequest(req *dto.CreateUserRequest) error {
	if err := validateUsername(req.Username); err != nil {
		return err
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if err := validateDisplayName(req.DisplayName); err != nil {
		return err
	}
	return validatePassword(req.Password)
}

func validateUpdateRequest(req *dto.UpdateUserRequest) error {
	if req.Email != nil {
		if *req.Email == "" {
			return errors.NewValidationError("email", "cannot be empty")
		}
		return validateEmail(*req.Email)
	}
	return nil
}

func validateUsername(username string) error {
	if err := validation.ValidateRequired("username", username); err != nil {
		return err
	}
	if !usernameRegex.MatchString(username) {
		return errors.NewValidationError("username", "must be 3-30 alphanumeric characters, underscore, or hyphen")
	}
	return nil
}

func validateEmail(email string) error {
	if err := validation.ValidateRequired("email", email); err != nil {
		return err
	}
	if !emailRegex.MatchString(email) {
		return errors.NewValidationError("email", "invalid format")
	}
	return nil
}

func validateDisplayName(displayName string) error {
	if err := validation.ValidateRequired("display_name", displayName); err != nil {
		return err
	}
	return validation.ValidateLength("display_name", displayName, minDisplayNameLength, maxDisplayNameLength)
}

func validatePassword(password string) error {
	if err := validation.ValidateRequired("password", password); err != nil {
		return err
	}
	if len(password) < minPasswordLength {
		return user.ErrWeakPassword.WithDetail("min_length", minPasswordLength)
	}
	if len(password) > maxPasswordLength {
		return errors.NewValidationError("password", "too long").WithDetail("max_length", maxPasswordLength)
	}
	if !letterRegex.MatchString(password) || !numberRegex.MatchString(password) {
		return user.ErrWeakPassword.WithDetail("requirement", "must contain letters and numbers")
	}
	return nil
}
