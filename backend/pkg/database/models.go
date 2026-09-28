package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func assignID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleHSEManager Role = "HSE_MANAGER"
	RoleHSEOfficer Role = "HSE_OFFICER"
	RoleSupervisor Role = "SUPERVISOR"
	RoleReporter   Role = "REPORTER"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email        string     `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string     `gorm:"size:255;not null"`
	Name         string     `gorm:"size:255;not null"`
	Role         Role       `gorm:"size:32;not null;index"`
	Status       UserStatus `gorm:"size:32;not null;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	SupervisedLocations []Location `gorm:"foreignKey:SupervisorID"`
	HSEOfficerLocations []Location `gorm:"foreignKey:HSEOfficerID"`
	ReportedIncidents   []Incident `gorm:"foreignKey:ReporterID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	assignID(&u.ID)
	return nil
}

type Region struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:255;not null"`
	Code      string    `gorm:"size:64;uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Locations []Location `gorm:"foreignKey:RegionID"`
}

func (r *Region) BeforeCreate(tx *gorm.DB) error {
	assignID(&r.ID)
	return nil
}

type LocationType string

const (
	LocationTambang   LocationType = "TAMBANG"
	LocationTelcoSite LocationType = "TELCO_SITE"
	LocationKantor    LocationType = "KANTOR"
	LocationGudang    LocationType = "GUDANG"
	LocationWorkshop  LocationType = "WORKSHOP"
	LocationLainnya   LocationType = "LAINNYA"
)

type Location struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey"`
	Name         string       `gorm:"size:255;not null"`
	Code         string       `gorm:"size:64;uniqueIndex;not null"`
	Type         LocationType `gorm:"size:32;not null"`
	RegionID     *uuid.UUID   `gorm:"type:uuid;index"`
	SupervisorID uuid.UUID    `gorm:"type:uuid;not null;index"`
	HSEOfficerID uuid.UUID    `gorm:"type:uuid;not null;index"`
	IsActive     bool         `gorm:"not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	Region     *Region `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Supervisor User    `gorm:"foreignKey:SupervisorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	HSEOfficer User    `gorm:"foreignKey:HSEOfficerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Areas      []Area  `gorm:"foreignKey:LocationID"`
	Incidents  []Incident
}

func (l *Location) BeforeCreate(tx *gorm.DB) error {
	assignID(&l.ID)
	return nil
}

type Area struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name       string    `gorm:"size:255;not null"`
	Code       string    `gorm:"size:64;not null"`
	LocationID uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Location  Location   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Incidents []Incident `gorm:"foreignKey:AreaID"`
}

func (a *Area) BeforeCreate(tx *gorm.DB) error {
	assignID(&a.ID)
	return nil
}

type IncidentCategory string

const (
	CategoryNearMiss         IncidentCategory = "NEAR_MISS"
	CategoryFirstAid         IncidentCategory = "FIRST_AID"
	CategoryMedicalTreatment IncidentCategory = "MEDICAL_TREATMENT"
	CategoryLTI              IncidentCategory = "LTI"
	CategoryFatality         IncidentCategory = "FATALITY"
	CategoryPropertyDamage   IncidentCategory = "PROPERTY_DAMAGE"
	CategoryEnvironmental    IncidentCategory = "ENVIRONMENTAL"
)

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type EscalationLevel string

const (
	EscalationL1 EscalationLevel = "L1"
	EscalationL2 EscalationLevel = "L2"
	EscalationL3 EscalationLevel = "L3"
)

type IncidentStatus string

const (
	StatusDraft              IncidentStatus = "DRAFT"
	StatusPendingReview      IncidentStatus = "PENDING_REVIEW"
	StatusUnderInvestigation IncidentStatus = "UNDER_INVESTIGATION"
	StatusCorrectiveAction   IncidentStatus = "CORRECTIVE_ACTION"
	StatusClosed             IncidentStatus = "CLOSED"
	StatusRejected           IncidentStatus = "REJECTED"
)

type Witness struct {
	Name     string `json:"name"`
	Position string `json:"position"`
}

type Incident struct {
	ID                uuid.UUID        `gorm:"type:uuid;primaryKey"`
	IncidentNumber    *string          `gorm:"size:32;uniqueIndex"`
	Title             string           `gorm:"size:255;not null"`
	Description       string           `gorm:"type:text;not null"`
	Category          IncidentCategory `gorm:"size:64;not null;index"`
	Severity          Severity         `gorm:"size:32;not null"`
	EscalationLevel   EscalationLevel  `gorm:"size:8;not null"`
	Status            IncidentStatus   `gorm:"size:32;not null;index"`
	IncidentDatetime  time.Time        `gorm:"not null;index"`
	LocationID        uuid.UUID        `gorm:"type:uuid;not null;index"`
	AreaID            *uuid.UUID       `gorm:"type:uuid;index"`
	ReporterID        uuid.UUID        `gorm:"type:uuid;not null;index"`
	HasVictim         bool             `gorm:"not null"`
	VictimName        *string          `gorm:"size:255"`
	VictimPosition    *string          `gorm:"size:255"`
	InjuryDescription *string          `gorm:"type:text"`
	InitialTreatment  *string          `gorm:"type:text"`
	Witnesses         []Witness        `gorm:"serializer:json"`
	PendingReviewAt   *time.Time
	ClosedAt          *time.Time
	ClosedByID        *uuid.UUID `gorm:"type:uuid"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	Location          Location `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Area              *Area    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Reporter          User     `gorm:"foreignKey:ReporterID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	ClosedBy          *User    `gorm:"foreignKey:ClosedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	WorkflowLogs      []IncidentWorkflowLog
	RCA               *RootCauseAnalysis
	CorrectiveActions []CorrectiveAction
	Files             []IncidentFile
}

func (i *Incident) BeforeCreate(tx *gorm.DB) error {
	assignID(&i.ID)
	return nil
}

// IncidentNumberCounter allocates INC-YYYY-MM-XXXX sequentially per calendar month (UTC).
type IncidentNumberCounter struct {
	Year    int `gorm:"primaryKey"`
	Month   int `gorm:"primaryKey"`
	LastSeq int `gorm:"not null"`
}

type IncidentWorkflowLog struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey"`
	IncidentID uuid.UUID      `gorm:"type:uuid;not null;index"`
	FromStatus IncidentStatus `gorm:"size:32;not null"`
	ToStatus   IncidentStatus `gorm:"size:32;not null"`
	ActorID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	Comment    *string        `gorm:"type:text"`
	CreatedAt  time.Time

	Incident Incident `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Actor    User     `gorm:"foreignKey:ActorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (l *IncidentWorkflowLog) BeforeCreate(tx *gorm.DB) error {
	assignID(&l.ID)
	return nil
}

type FiveWhy struct {
	Why    string `json:"why"`
	Answer string `json:"answer"`
}

type Fishbone struct {
	Man         string `json:"man"`
	Machine     string `json:"machine"`
	Method      string `json:"method"`
	Material    string `json:"material"`
	Environment string `json:"environment"`
	Measurement string `json:"measurement"`
}

type RootCauseAnalysis struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	IncidentID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	Timeline          string    `gorm:"type:text"`
	HumanFactor       string    `gorm:"type:text"`
	EnvironmentFactor string    `gorm:"type:text"`
	EquipmentFactor   string    `gorm:"type:text"`
	FiveWhys          []FiveWhy `gorm:"serializer:json"`
	Fishbone          Fishbone  `gorm:"serializer:json"`
	InvestigatorID    uuid.UUID `gorm:"type:uuid;not null;index"`
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time

	Incident     Incident `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Investigator User     `gorm:"foreignKey:InvestigatorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (r *RootCauseAnalysis) BeforeCreate(tx *gorm.DB) error {
	assignID(&r.ID)
	return nil
}

type ActionType string

const (
	ActionImmediate ActionType = "IMMEDIATE"
	ActionShortTerm ActionType = "SHORT_TERM"
	ActionLongTerm  ActionType = "LONG_TERM"
)

type CAPriority string

const (
	PriorityLow    CAPriority = "LOW"
	PriorityMedium CAPriority = "MEDIUM"
	PriorityHigh   CAPriority = "HIGH"
)

type CAStatus string

const (
	CAStatusOpen       CAStatus = "OPEN"
	CAStatusInProgress CAStatus = "IN_PROGRESS"
	CAStatusDone       CAStatus = "DONE"
	CAStatusOverdue    CAStatus = "OVERDUE"
	CAStatusVerified   CAStatus = "VERIFIED"
)

type CorrectiveAction struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	IncidentID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	Description     string     `gorm:"type:text;not null"`
	ActionType      ActionType `gorm:"size:32;not null"`
	Priority        CAPriority `gorm:"size:32;not null"`
	Status          CAStatus   `gorm:"size:32;not null;index"`
	AssigneeID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	DueDate         time.Time  `gorm:"type:date;not null;index"`
	CompletionNotes *string    `gorm:"type:text"`
	CompletedAt     *time.Time
	VerifiedByID    *uuid.UUID `gorm:"type:uuid"`
	VerifiedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Incident   Incident `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Assignee   User     `gorm:"foreignKey:AssigneeID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	VerifiedBy *User    `gorm:"foreignKey:VerifiedByID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Files      []IncidentFile
}

func (c *CorrectiveAction) BeforeCreate(tx *gorm.DB) error {
	assignID(&c.ID)
	return nil
}

type FileContext string

const (
	FileIncidentEvidence FileContext = "INCIDENT_EVIDENCE"
	FileRCAEvidence      FileContext = "RCA_EVIDENCE"
	FileCACompletion     FileContext = "CA_COMPLETION"
	FileReport           FileContext = "REPORT"
)

type IncidentFile struct {
	ID                 uuid.UUID   `gorm:"type:uuid;primaryKey"`
	IncidentID         uuid.UUID   `gorm:"type:uuid;not null;index"`
	CorrectiveActionID *uuid.UUID  `gorm:"type:uuid;index"`
	UploadedByID       uuid.UUID   `gorm:"type:uuid;not null;index"`
	OriginalName       string      `gorm:"size:512;not null"`
	StoredKey          string      `gorm:"size:512;not null"`
	MimeType           string      `gorm:"size:128;not null"`
	SizeBytes          int64       `gorm:"not null"`
	Context            FileContext `gorm:"size:32;not null"`
	CreatedAt          time.Time

	Incident         Incident          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CorrectiveAction *CorrectiveAction `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	UploadedBy       User              `gorm:"foreignKey:UploadedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (f *IncidentFile) BeforeCreate(tx *gorm.DB) error {
	assignID(&f.ID)
	return nil
}

type NotificationType string

const (
	NotifIncidentSubmitted NotificationType = "INCIDENT_SUBMITTED"
	NotifIncidentEscalated NotificationType = "INCIDENT_ESCALATED"
	NotifSLAWarning        NotificationType = "SLA_WARNING"
	NotifCAAssigned        NotificationType = "CA_ASSIGNED"
	NotifCAOverdue         NotificationType = "CA_OVERDUE"
)

type NotificationPriority string

const (
	NotifInfo     NotificationPriority = "INFO"
	NotifMedium   NotificationPriority = "MEDIUM"
	NotifHigh     NotificationPriority = "HIGH"
	NotifCritical NotificationPriority = "CRITICAL"
)

type Notification struct {
	ID            uuid.UUID            `gorm:"type:uuid;primaryKey"`
	RecipientID   uuid.UUID            `gorm:"type:uuid;not null;index"`
	Type          NotificationType     `gorm:"size:64;not null;index"`
	Title         string               `gorm:"size:255;not null"`
	Body          string               `gorm:"type:text;not null"`
	Priority      NotificationPriority `gorm:"size:32;not null"`
	IsRead        bool                 `gorm:"not null;default:false"`
	ReferenceType string               `gorm:"size:64;not null"`
	ReferenceID   uuid.UUID            `gorm:"type:uuid;not null;index"`
	CreatedAt     time.Time

	Recipient User `gorm:"foreignKey:RecipientID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	assignID(&n.ID)
	return nil
}

type AuditAction string

const (
	AuditCreated       AuditAction = "CREATED"
	AuditUpdated       AuditAction = "UPDATED"
	AuditStatusChanged AuditAction = "STATUS_CHANGED"
	AuditFileUploaded  AuditAction = "FILE_UPLOADED"
	AuditApproved      AuditAction = "APPROVED"
	AuditRejected      AuditAction = "REJECTED"
	AuditClosed        AuditAction = "CLOSED"
)

type AuditLog struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index"`
	UserRole   string         `gorm:"size:32;not null"`
	IPAddress  string         `gorm:"size:64;not null"`
	EntityType string         `gorm:"size:64;not null;index"`
	EntityID   string         `gorm:"size:64;not null;index"`
	Action     AuditAction    `gorm:"size:32;not null;index"`
	Before     map[string]any `gorm:"serializer:json"`
	After      map[string]any `gorm:"serializer:json"`
	CreatedAt  time.Time

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	assignID(&a.ID)
	return nil
}
