package domain

import "time"

// EventStatus represents the lifecycle of an event.
type EventStatus string

const (
	EventStatusUpcoming  EventStatus = "UPCOMING"
	EventStatusOnSale    EventStatus = "ON_SALE"
	EventStatusSoldOut   EventStatus = "SOLD_OUT"
	EventStatusCancelled EventStatus = "CANCELLED"
)

// Event is the canonical record for a show/concert/match.
type Event struct {
	EventID        int64       `json:"event_id"`
	EventName      string      `json:"event_name"`
	EventDate      time.Time   `json:"event_date"`
	VenueName      string      `json:"venue_name"`
	TotalSeats     int         `json:"total_seats"`
	AvailableSeats int         `json:"available_seats"`
	Status         EventStatus `json:"status"`
	SaleStartTime  *time.Time  `json:"sale_start_time,omitempty"`
	Version        int64       `json:"version"` // optimistic locking
	CreatedAt      time.Time   `json:"created_at"`
}

// SeatStatus represents the lifecycle of a single seat.
type SeatStatus string

const (
	SeatStatusAvailable SeatStatus = "AVAILABLE"
	SeatStatusReserved  SeatStatus = "RESERVED"
	SeatStatusBooked    SeatStatus = "BOOKED"
	SeatStatusBlocked   SeatStatus = "BLOCKED"
)

// SeatType categorises the tier of a seat.
type SeatType string

const (
	SeatTypeRegular SeatType = "REGULAR"
	SeatTypeVIP     SeatType = "VIP"
	SeatTypePremium SeatType = "PREMIUM"
)

// Seat is the live inventory record for one physical seat.
// The `version` column enables optimistic locking.
// CONCURRENCY: This row is the contention point. All seat mutations
// MUST go through a distributed lock (LockManager) keyed on
// "seat:<event_id>:<seat_number>" before touching this record.
type Seat struct {
	SeatID        int64      `json:"seat_id"`
	EventID       int64      `json:"event_id"`
	SeatNumber    string     `json:"seat_number"`
	Section       string     `json:"section"`
	RowNumber     string     `json:"row_number"`
	SeatType      SeatType   `json:"seat_type"`
	Price         float64    `json:"price"`
	Status        SeatStatus `json:"status"`
	Version       int64      `json:"version"` // optimistic locking
	ReservedBy    *string    `json:"reserved_by,omitempty"`
	ReservedUntil *time.Time `json:"reserved_until,omitempty"`
	BookingID     *int64     `json:"booking_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// BookingStatus represents the state of a customer order.
type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "PENDING"
	BookingStatusConfirmed BookingStatus = "CONFIRMED"
	BookingStatusCancelled BookingStatus = "CANCELLED"
	BookingStatusFailed    BookingStatus = "FAILED"
)

// PaymentStatus represents the state of the payment leg.
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

// Booking is the customer order header after successful payment.
type Booking struct {
	BookingID        int64         `json:"booking_id"`
	EventID          int64         `json:"event_id"`
	UserID           string        `json:"user_id"`
	TotalAmount      float64       `json:"total_amount"`
	Status           BookingStatus `json:"status"`
	PaymentID        *string       `json:"payment_id,omitempty"`
	PaymentStatus    PaymentStatus `json:"payment_status"`
	BookingReference string        `json:"booking_reference"`
	CreatedAt        time.Time     `json:"created_at"`
	ConfirmedAt      *time.Time    `json:"confirmed_at,omitempty"`
}

// BookingSeat is the junction between a booking and one seat,
// capturing the price at purchase time (supports dynamic pricing).
type BookingSeat struct {
	BookingSeatID int64   `json:"booking_seat_id"`
	BookingID     int64   `json:"booking_id"`
	SeatID        int64   `json:"seat_id"`
	Price         float64 `json:"price"`
}

// ReservationStatus represents the state of a temporary hold.
type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "ACTIVE"
	ReservationStatusConfirmed ReservationStatus = "CONFIRMED"
	ReservationStatusExpired   ReservationStatus = "EXPIRED"
	ReservationStatusCancelled ReservationStatus = "CANCELLED"
)

// Reservation is an ephemeral hold while the user completes payment.
// CONCURRENCY: Reservations are created inside the distributed lock.
// The cleanup goroutine releases expired reservations every minute.
type Reservation struct {
	ReservationID int64             `json:"reservation_id"`
	SeatID        int64             `json:"seat_id"`
	EventID       int64             `json:"event_id"`
	UserID        string            `json:"user_id"`
	SessionID     *string           `json:"session_id,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at"`
	Status        ReservationStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
}

// User is the auth identity record.
type User struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // USER | ADMIN
	CreatedAt    time.Time `json:"created_at"`
}
