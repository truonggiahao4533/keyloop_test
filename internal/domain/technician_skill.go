package domain

import "time"

type TechnicianSkill struct {
	TechnicianID string
	Skill        ServiceType
	DeletedAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
