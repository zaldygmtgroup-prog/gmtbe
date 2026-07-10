package models

import "time"

type PreorderStatus string
type PaymentStatus string
type PaymentMode string
type PaymentStage string

const (
	PreorderStatusDraft    PreorderStatus = "draft"
	PreorderStatusInReview PreorderStatus = "in_review"
	PreorderStatusApprove  PreorderStatus = "approve"
	PreorderStatusInvalid  PreorderStatus = "invalid"

	PaymentStatusUnpaid  PaymentStatus = "unpaid"
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPartial PaymentStatus = "partial"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusExpired PaymentStatus = "expired"
	PaymentStatusFailed  PaymentStatus = "failed"
	PaymentStatusRefund  PaymentStatus = "refund"

	PaymentModeFull  PaymentMode = "full"
	PaymentModeSplit PaymentMode = "split"

	PaymentStageFull      PaymentStage = "full"
	PaymentStageDP        PaymentStage = "dp"
	PaymentStageRemaining PaymentStage = "remaining"
)

type Preorder struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	PONumber         string         `gorm:"size:50;uniqueIndex;column:po_number" json:"po_number"`
	IDAgent          uint           `gorm:"column:id_agent;index;not null" json:"id_agent"`
	Agent            *User          `gorm:"foreignKey:IDAgent;constraint:OnDelete:CASCADE" json:"agent,omitempty"`
	NamaCustomer     string         `gorm:"size:255;not null;column:nama_customer" json:"nama_customer"`
	NamaPerusahaan   string         `gorm:"size:255;column:nama_perusahaan" json:"nama_perusahaan,omitempty"`
	Email            string         `gorm:"size:255;not null" json:"email"`
	Alamat           string         `gorm:"type:text;not null" json:"alamat"`
	NoHP             string         `gorm:"size:50;not null;column:no_hp" json:"no_hp"`
	Catatan          string         `gorm:"type:text" json:"catatan"`
	Subtotal         int64          `gorm:"not null" json:"subtotal"`
	TotalDiscount    int64          `gorm:"not null;column:total_discount" json:"total_discount"`
	Total            int64          `gorm:"not null" json:"total"`
	TotalKomisi      int64          `gorm:"not null;column:total_komisi" json:"total_komisi"`
	Status           PreorderStatus `gorm:"type:enum('draft','in_review','approve','invalid');default:'draft';not null" json:"status"`
	PaymentMode      PaymentMode    `gorm:"type:enum('full','split');default:'full';not null;column:payment_mode" json:"payment_mode"`
	PaymentStatus    PaymentStatus  `gorm:"type:enum('unpaid','pending','partial','paid','expired','failed','refund');default:'unpaid';not null;column:payment_status" json:"payment_status"`
	PaymentProof     string         `gorm:"size:500;column:payment_proof" json:"payment_proof,omitempty"`
	DPProof          string         `gorm:"size:500;column:dp_proof" json:"dp_proof,omitempty"`
	RemainingProof   string         `gorm:"size:500;column:remaining_proof" json:"remaining_proof,omitempty"`
	LastPaymentStage string         `gorm:"size:20;column:last_payment_stage" json:"last_payment_stage,omitempty"`
	InvalidReason    *string        `gorm:"type:text;column:invalid_reason" json:"invalid_reason,omitempty"`
	Items            []PreorderItem `gorm:"foreignKey:IDPreorder;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type PreorderItem struct {
	ID                         uint      `gorm:"primaryKey" json:"id"`
	IDPreorder                 uint      `gorm:"column:id_preorder;index;not null" json:"id_preorder"`
	IDProduct                  uint      `gorm:"column:id_product;index;not null" json:"id_product"`
	Product                    *Product  `gorm:"foreignKey:IDProduct;constraint:OnDelete:RESTRICT" json:"product,omitempty"`
	ProductNameSnapshot        string    `gorm:"size:255;not null;column:product_name_snapshot" json:"namaproduct"`
	ProductPhotoSnapshot       string    `gorm:"size:255;column:product_photo_snapshot" json:"foto"`
	ProductDescriptionSnapshot string    `gorm:"type:text;column:product_description_snapshot" json:"deskripsi"`
	UnitSnapshot               string    `gorm:"size:50;column:unit_snapshot" json:"unit"`
	UnitPrice                  int64     `gorm:"not null;column:unit_price" json:"unit_price"`
	Qty                        int       `gorm:"not null" json:"qty"`
	DiscountPercent            float64   `gorm:"not null;column:discount_percent" json:"discount_percent"`
	DiscountAmount             int64     `gorm:"not null;column:discount_amount" json:"discount_amount"`
	Subtotal                   int64     `gorm:"not null" json:"subtotal"`
	Total                      int64     `gorm:"not null" json:"total"`
	Komisi                     int64     `gorm:"not null;column:komisi" json:"komisi"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

func IsValidPreorderStatus(status PreorderStatus) bool {
	switch status {
	case PreorderStatusDraft, PreorderStatusInReview, PreorderStatusApprove, PreorderStatusInvalid:
		return true
	default:
		return false
	}
}
