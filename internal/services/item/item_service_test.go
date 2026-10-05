package item

import (
	"context"
	"errors"
	"testing"

	"checkout-api/internal/domain"
	filter "checkout-api/internal/helper/filter"
	itemrepo "checkout-api/internal/repository/item"
	"checkout-api/internal/services/item/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestItemService_GetAllItems(t *testing.T) {
	expectedItems := []*domain.Item{
		{
			ID:    1,
			Name:  "Laptop",
			Price: 100000,
			Stock: 10,
		},
		{
			ID:    2,
			Name:  "Phone",
			Price: 50000,
			Stock: 20,
		},
	}

	tests := []struct {
		name      string
		wantItems []*domain.Item
		wantErr   bool
		setupMock func(*mocks.MockItemStore)
	}{
		{
			name: "success",
			setupMock: func(store *mocks.MockItemStore) {
				store.EXPECT().
					GetItems(gomock.Any(), filter.Request{}).
					Return(expectedItems, nil).
					Times(1)
			},
			wantItems: expectedItems,
			wantErr:   false,
		},
		{
			name: "store error",
			setupMock: func(store *mocks.MockItemStore) {
				store.EXPECT().
					GetItems(gomock.Any(), filter.Request{}).
					Return(nil, errors.New("database unavailable")).
					Times(1)
			},
			wantItems: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mocks.NewMockItemStore(ctrl)

			tt.setupMock(mockStore)

			service := NewItemService(mockStore)

			got, err := service.GetAllItems(
				context.Background(),
				filter.Request{},
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"GetAllItems() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if len(got) != len(tt.wantItems) {
				t.Fatalf(
					"GetAllItems() returned %d items, want %d",
					len(got),
					len(tt.wantItems),
				)
			}
		})
	}
}

func TestItemService_GetItemByID(t *testing.T) {
	expectedItem := &domain.Item{
		ID:    1,
		Name:  "Laptop",
		Price: 100000,
		Stock: 10,
	}

	tests := []struct {
		name       string
		id         int
		wantItem   *domain.Item
		wantErr    bool
		expectedIs error
		setupMock  func(*mocks.MockItemStore)
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(store *mocks.MockItemStore) {
				store.EXPECT().
					GetItemByID(gomock.Any(), 1).
					Return(expectedItem, nil).
					Times(1)
			},
			wantItem: expectedItem,
			wantErr:  false,
		},
		{
			name: "invalid id",
			id:   0,
			setupMock: func(store *mocks.MockItemStore) {
				// No EXPECT().
				// Store should not be called for invalid ID.
			},
			wantItem:   nil,
			wantErr:    true,
			expectedIs: ErrItemNotFound,
		},
		{
			name: "item not found",
			id:   999,
			setupMock: func(store *mocks.MockItemStore) {
				store.EXPECT().
					GetItemByID(gomock.Any(), 999).
					Return(nil, itemrepo.ErrItemNotFound).
					Times(1)
			},
			wantItem:   nil,
			wantErr:    true,
			expectedIs: ErrItemNotFound,
		},
		{
			name: "unexpected store error",
			id:   1,
			setupMock: func(store *mocks.MockItemStore) {
				store.EXPECT().
					GetItemByID(gomock.Any(), 1).
					Return(nil, errors.New("database connection failed")).
					Times(1)
			},
			wantItem: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mocks.NewMockItemStore(ctrl)

			tt.setupMock(mockStore)

			service := NewItemService(mockStore)

			got, err := service.GetItemByID(
				context.Background(),
				tt.id,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"GetItemByID() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if tt.expectedIs != nil {
				if !errors.Is(err, tt.expectedIs) {
					t.Fatalf(
						"GetItemByID() error = %v, want errors.Is(..., %v)",
						err,
						tt.expectedIs,
					)
				}
			}

			if got != tt.wantItem {
				t.Fatalf(
					"GetItemByID() returned %v, want %v",
					got,
					tt.wantItem,
				)
			}
		})
	}
}

func TestNewItemService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockItemStore(ctrl)

	service := NewItemService(mockStore)

	assert.NotNil(t, service)
	assert.Equal(t, mockStore, service.store)
}
