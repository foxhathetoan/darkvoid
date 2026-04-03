package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jarviisha/darkvoid/internal/feature/user/db"
	"github.com/jarviisha/darkvoid/internal/feature/user/dto"
	"github.com/jarviisha/darkvoid/internal/feature/user/entity"
	"github.com/jarviisha/darkvoid/pkg/errors"
)

// --------------------------------------------------------------------------
// Mock: userRepo
// --------------------------------------------------------------------------

type mockUserRepo struct {
	existsUsername       func(ctx context.Context, username string) (bool, error)
	existsEmail          func(ctx context.Context, email string) (bool, error)
	existsEmailExcluding func(ctx context.Context, email string, userID uuid.UUID) (bool, error)
	createUser           func(ctx context.Context, user *entity.User) (*entity.User, error)
	getUserByID          func(ctx context.Context, id uuid.UUID) (*entity.User, error)
	getUserByUsername    func(ctx context.Context, username string) (*entity.User, error)
	getUserByEmail       func(ctx context.Context, email string) (*entity.User, error)
	updateUser           func(ctx context.Context, id uuid.UUID, email *string, updatedBy *uuid.UUID) (*entity.User, error)
	updateUserProfile    func(ctx context.Context, id uuid.UUID, params db.UpdateUserProfileParams) (*entity.User, error)
	updateUserPassword   func(ctx context.Context, id uuid.UUID, passwordHash string, updatedBy *uuid.UUID) error
	deactivateUser       func(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error
}

func (m *mockUserRepo) ExistsUsername(ctx context.Context, username string) (bool, error) {
	if m.existsUsername != nil {
		return m.existsUsername(ctx, username)
	}
	return false, nil
}
func (m *mockUserRepo) ExistsEmail(ctx context.Context, email string) (bool, error) {
	if m.existsEmail != nil {
		return m.existsEmail(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepo) ExistsEmailExcludingUser(ctx context.Context, email string, userID uuid.UUID) (bool, error) {
	if m.existsEmailExcluding != nil {
		return m.existsEmailExcluding(ctx, email, userID)
	}
	return false, nil
}
func (m *mockUserRepo) CreateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	if m.createUser != nil {
		return m.createUser(ctx, user)
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	return user, nil
}
func (m *mockUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, id)
	}
	return nil, errors.ErrNotFound
}
func (m *mockUserRepo) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	if m.getUserByUsername != nil {
		return m.getUserByUsername(ctx, username)
	}
	return nil, errors.ErrNotFound
}
func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.getUserByEmail != nil {
		return m.getUserByEmail(ctx, email)
	}
	return nil, errors.ErrNotFound
}
func (m *mockUserRepo) UpdateUser(ctx context.Context, id uuid.UUID, email *string, updatedBy *uuid.UUID) (*entity.User, error) {
	if m.updateUser != nil {
		return m.updateUser(ctx, id, email, updatedBy)
	}
	return nil, nil //nolint:nilnil // mock returns zero values
}
func (m *mockUserRepo) UpdateUserProfile(ctx context.Context, id uuid.UUID, params db.UpdateUserProfileParams) (*entity.User, error) {
	if m.updateUserProfile != nil {
		return m.updateUserProfile(ctx, id, params)
	}
	return activeUser(id), nil
}
func (m *mockUserRepo) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string, updatedBy *uuid.UUID) error {
	if m.updateUserPassword != nil {
		return m.updateUserPassword(ctx, id, passwordHash, updatedBy)
	}
	return nil
}
func (m *mockUserRepo) DeactivateUser(ctx context.Context, id uuid.UUID, updatedBy *uuid.UUID) error {
	if m.deactivateUser != nil {
		return m.deactivateUser(ctx, id, updatedBy)
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func newUserService(repo userRepo) *UserService {
	return &UserService{userRepo: repo, storage: nil}
}

func validCreateReq() *dto.CreateUserRequest {
	return &dto.CreateUserRequest{
		Username:    "johndoe",
		Email:       "john@example.com",
		DisplayName: "John Doe",
		Password:    "SecurePass123",
	}
}

func activeUser(id uuid.UUID) *entity.User {
	return &entity.User{
		ID:          id,
		Username:    "johndoe",
		Email:       "john@example.com",
		IsActive:    true,
		DisplayName: "John Doe",
		CreatedAt:   time.Now(),
	}
}

// --------------------------------------------------------------------------
// CreateUser tests
// --------------------------------------------------------------------------

func TestCreateUser_Success(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserRepo{
		createUser: func(_ context.Context, u *entity.User) (*entity.User, error) {
			u.ID = userID
			return u, nil
		},
	}

	svc := newUserService(repo)
	id, err := svc.CreateUser(context.Background(), validCreateReq())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != userID {
		t.Errorf("expected id %v, got %v", userID, id)
	}
}

func TestCreateUser_InvalidUsername_TooShort(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	req := validCreateReq()
	req.Username = "ab" // less than 3 chars

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertServiceErrorCode(t, err, "VALIDATION_ERROR")
}

func TestCreateUser_InvalidUsername_InvalidChars(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	req := validCreateReq()
	req.Username = "john doe" // space not allowed

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertServiceErrorCode(t, err, "VALIDATION_ERROR")
}

func TestCreateUser_InvalidEmail(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	req := validCreateReq()
	req.Email = "not-an-email"

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertServiceErrorCode(t, err, "VALIDATION_ERROR")
}

func TestCreateUser_WeakPassword_TooShort(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	req := validCreateReq()
	req.Password = "abc123" // 6 chars, below minimum 8

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Fatal("expected weak password error, got nil")
	}
	assertServiceErrorCode(t, err, "WEAK_PASSWORD")
}

func TestCreateUser_WeakPassword_NoNumber(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	req := validCreateReq()
	req.Password = "OnlyLetters"

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Fatal("expected weak password error, got nil")
	}
	assertServiceErrorCode(t, err, "WEAK_PASSWORD")
}

func TestCreateUser_WeakPassword_NoLetter(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	req := validCreateReq()
	req.Password = "12345678"

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Fatal("expected weak password error, got nil")
	}
	assertServiceErrorCode(t, err, "WEAK_PASSWORD")
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	repo := &mockUserRepo{
		existsUsername: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newUserService(repo)

	_, err := svc.CreateUser(context.Background(), validCreateReq())
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	assertServiceErrorCode(t, err, "CONFLICT")
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{
		existsEmail: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newUserService(repo)

	_, err := svc.CreateUser(context.Background(), validCreateReq())
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	assertServiceErrorCode(t, err, "CONFLICT")
}

func TestCreateUser_EmailNormalized(t *testing.T) {
	var savedEmail string
	repo := &mockUserRepo{
		createUser: func(_ context.Context, u *entity.User) (*entity.User, error) {
			savedEmail = u.Email
			u.ID = uuid.New()
			return u, nil
		},
	}
	svc := newUserService(repo)
	req := validCreateReq()
	req.Email = "JOHN@EXAMPLE.COM"

	_, err := svc.CreateUser(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedEmail != "john@example.com" {
		t.Errorf("expected email to be lowercased to john@example.com, got %q", savedEmail)
	}
}

func TestCreateUser_DisplayNameSaved(t *testing.T) {
	var savedDisplayName string
	repo := &mockUserRepo{
		createUser: func(_ context.Context, u *entity.User) (*entity.User, error) {
			savedDisplayName = u.DisplayName
			u.ID = uuid.New()
			return u, nil
		},
	}
	svc := newUserService(repo)
	req := validCreateReq()
	req.DisplayName = "John Doe"

	_, err := svc.CreateUser(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedDisplayName != "John Doe" {
		t.Errorf("expected display name %q, got %q", "John Doe", savedDisplayName)
	}
}

// --------------------------------------------------------------------------
// GetUserByID tests
// --------------------------------------------------------------------------

func TestGetUserByID_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(id), nil
		},
	}
	svc := newUserService(repo)

	user, err := svc.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != id {
		t.Errorf("expected id %v, got %v", id, user.ID)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.ErrNotFound
		},
	}
	svc := newUserService(repo)

	_, err := svc.GetUserByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "USER_NOT_FOUND")
}

func TestGetUserByID_InactiveUser(t *testing.T) {
	id := uuid.New()
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(id)
			u.IsActive = false
			return u, nil
		},
	}
	svc := newUserService(repo)

	_, err := svc.GetUserByID(context.Background(), id)
	if err == nil {
		t.Fatal("expected error for inactive user, got nil")
	}
	assertServiceErrorCode(t, err, "ACCOUNT_DISABLED")
}

// --------------------------------------------------------------------------
// GetUserByUsername tests
// --------------------------------------------------------------------------

func TestGetUserByUsername_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockUserRepo{
		getUserByUsername: func(_ context.Context, _ string) (*entity.User, error) {
			return activeUser(id), nil
		},
	}
	svc := newUserService(repo)

	user, err := svc.GetUserByUsername(context.Background(), "johndoe")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != id {
		t.Errorf("expected id %v, got %v", id, user.ID)
	}
}

func TestGetUserByUsername_EmptyUsername(t *testing.T) {
	svc := newUserService(&mockUserRepo{})

	_, err := svc.GetUserByUsername(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error for empty username, got nil")
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		getUserByUsername: func(_ context.Context, _ string) (*entity.User, error) {
			return nil, errors.ErrNotFound
		},
	}
	svc := newUserService(repo)

	_, err := svc.GetUserByUsername(context.Background(), "nobody")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "USER_NOT_FOUND")
}

// --------------------------------------------------------------------------
// GetUserByEmail tests
// --------------------------------------------------------------------------

func TestGetUserByEmail_EmptyEmail(t *testing.T) {
	svc := newUserService(&mockUserRepo{})

	_, err := svc.GetUserByEmail(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error for empty email, got nil")
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		getUserByEmail: func(_ context.Context, _ string) (*entity.User, error) {
			return nil, errors.ErrNotFound
		},
	}
	svc := newUserService(repo)

	_, err := svc.GetUserByEmail(context.Background(), "nobody@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "USER_NOT_FOUND")
}

// --------------------------------------------------------------------------
// UpdateUser tests
// --------------------------------------------------------------------------

func TestUpdateUser_Success(t *testing.T) {
	id := uuid.New()
	newEmail := "new@example.com"
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(id), nil
		},
		updateUser: func(_ context.Context, _ uuid.UUID, email *string, _ *uuid.UUID) (*entity.User, error) {
			u := activeUser(id)
			u.Email = *email
			return u, nil
		},
	}
	svc := newUserService(repo)

	user, err := svc.UpdateUser(context.Background(), id, &dto.UpdateUserRequest{Email: &newEmail}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != newEmail {
		t.Errorf("expected email %q, got %q", newEmail, user.Email)
	}
}

func TestUpdateUser_InvalidEmail(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	bad := "not-an-email"

	_, err := svc.UpdateUser(context.Background(), uuid.New(), &dto.UpdateUserRequest{Email: &bad}, nil)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertServiceErrorCode(t, err, "VALIDATION_ERROR")
}

func TestUpdateUser_EmptyEmail(t *testing.T) {
	svc := newUserService(&mockUserRepo{})
	empty := ""

	_, err := svc.UpdateUser(context.Background(), uuid.New(), &dto.UpdateUserRequest{Email: &empty}, nil)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	assertServiceErrorCode(t, err, "VALIDATION_ERROR")
}

func TestUpdateUser_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.ErrNotFound
		},
	}
	svc := newUserService(repo)
	email := "new@example.com"

	_, err := svc.UpdateUser(context.Background(), uuid.New(), &dto.UpdateUserRequest{Email: &email}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "USER_NOT_FOUND")
}

func TestUpdateUser_DuplicateEmail(t *testing.T) {
	id := uuid.New()
	email := "taken@example.com"
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(id), nil
		},
		existsEmailExcluding: func(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	svc := newUserService(repo)

	_, err := svc.UpdateUser(context.Background(), id, &dto.UpdateUserRequest{Email: &email}, nil)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	assertServiceErrorCode(t, err, "CONFLICT")
}

func TestUpdateUser_SameEmail_NoUniquenessCheck(t *testing.T) {
	id := uuid.New()
	sameEmail := "john@example.com"
	uniquenessCheckCalled := false
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(id), nil // activeUser has email john@example.com
		},
		existsEmailExcluding: func(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
			uniquenessCheckCalled = true
			return false, nil
		},
		updateUser: func(_ context.Context, _ uuid.UUID, email *string, _ *uuid.UUID) (*entity.User, error) {
			return activeUser(id), nil
		},
	}
	svc := newUserService(repo)

	_, err := svc.UpdateUser(context.Background(), id, &dto.UpdateUserRequest{Email: &sameEmail}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uniquenessCheckCalled {
		t.Error("uniqueness check should not be called when email is unchanged")
	}
}

// --------------------------------------------------------------------------
// DeactivateUser tests
// --------------------------------------------------------------------------

func TestDeactivateUser_Success(t *testing.T) {
	id := uuid.New()
	deactivateCalled := false
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return activeUser(id), nil
		},
		deactivateUser: func(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
			deactivateCalled = true
			return nil
		},
	}
	svc := newUserService(repo)

	err := svc.DeactivateUser(context.Background(), id, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deactivateCalled {
		t.Error("expected DeactivateUser to be called on repository")
	}
}

func TestDeactivateUser_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			return nil, errors.ErrNotFound
		},
	}
	svc := newUserService(repo)

	err := svc.DeactivateUser(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "USER_NOT_FOUND")
}

func TestDeactivateUser_AlreadyInactive(t *testing.T) {
	id := uuid.New()
	repo := &mockUserRepo{
		getUserByID: func(_ context.Context, _ uuid.UUID) (*entity.User, error) {
			u := activeUser(id)
			u.IsActive = false
			return u, nil
		},
	}
	svc := newUserService(repo)

	err := svc.DeactivateUser(context.Background(), id, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "ACCOUNT_DISABLED")
}

// --------------------------------------------------------------------------
// BootstrapRootUser tests
// --------------------------------------------------------------------------

func TestBootstrapRootUser_CreatesWhenEmpty(t *testing.T) {
	created := false
	repo := &mockUserRepo{
		createUser: func(_ context.Context, u *entity.User) (*entity.User, error) {
			created = true
			u.ID = uuid.New()
			return u, nil
		},
	}
	svc := newUserService(repo)

	ok, err := svc.BootstrapRootUser(context.Background(), "root@example.com", "RootPass123", "root", "Root User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected bootstrap to return true")
	}
	if !created {
		t.Error("expected CreateUser to be called")
	}
}

func TestBootstrapRootUser_SkipsWhenUsersExist(t *testing.T) {
	repo := &mockUserRepo{
		existsUsername: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newUserService(repo)

	ok, err := svc.BootstrapRootUser(context.Background(), "root@example.com", "RootPass123", "root", "Root User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected bootstrap to return false (skip) when users exist")
	}
}

// --------------------------------------------------------------------------
// UpdateMyProfile tests
// --------------------------------------------------------------------------

func TestUpdateMyProfile_Success(t *testing.T) {
	id := uuid.New()
	bio := "Software engineer"
	updateCalled := false
	repo := &mockUserRepo{
		updateUserProfile: func(_ context.Context, _ uuid.UUID, params db.UpdateUserProfileParams) (*entity.User, error) {
			updateCalled = true
			u := activeUser(id)
			u.Bio = params.Bio
			return u, nil
		},
	}
	svc := newUserService(repo)

	u, err := svc.UpdateMyProfile(context.Background(), id, &dto.UpdateProfileRequest{Bio: &bio})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !updateCalled {
		t.Error("expected UpdateUserProfile to be called")
	}
	if u.Bio == nil || *u.Bio != bio {
		t.Errorf("expected bio %q, got %v", bio, u.Bio)
	}
}

func TestUpdateMyProfile_NotFound(t *testing.T) {
	repo := &mockUserRepo{
		updateUserProfile: func(_ context.Context, _ uuid.UUID, _ db.UpdateUserProfileParams) (*entity.User, error) {
			return nil, errors.ErrNotFound
		},
	}
	svc := newUserService(repo)

	_, err := svc.UpdateMyProfile(context.Background(), uuid.New(), &dto.UpdateProfileRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServiceErrorCode(t, err, "USER_NOT_FOUND")
}

// --------------------------------------------------------------------------
// Helper: assert error code
// --------------------------------------------------------------------------

func assertServiceErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	appErr := errors.GetAppError(err)
	if appErr == nil {
		t.Fatalf("expected AppError with code %q, got non-AppError: %v", code, err)
	}
	if appErr.Code != code {
		t.Errorf("expected error code %q, got %q", code, appErr.Code)
	}
}
