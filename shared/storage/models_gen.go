package storage

import (
	"time"

	"uuid"

	"gorm.io/gorm"
)

type KaemanUser struct {
	UID      uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Nickname string    `gorm:"not null"`

	Identities []KaemanIdentity `gorm:"foreignKey:UID;references:UID"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

type KaemanIdentity struct {
	UID uuid.UUID `gorm:"type:uuid;not null;index"`

	Platform Platform `gorm:"not null;primaryKey"`
	Identity string   `gorm:"not null;primaryKey"`

	BindFrom  BindFrom `gorm:"not null"`
	CreatedAt time.Time
}
