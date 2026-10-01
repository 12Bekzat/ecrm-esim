package httpapi

import "time"

type User struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Login       string     `json:"login"`
	Role        string     `json:"role"`
	Permissions []string   `json:"permissions"`
	Status      string     `json:"status"`
	LastSeenAt  *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Customer struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Phone     string     `json:"phone"`
	Email     string     `json:"email"`
	Country   string     `json:"country"`
	Status    string     `json:"status"`
	Notes     string     `json:"notes"`
	Orders    int        `json:"orders"`
	Total     float64    `json:"total"`
	Profit    float64    `json:"profit"`
	LastOrder *time.Time `json:"lastOrderAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Supplier struct {
	ID                  string     `json:"id"`
	Code                string     `json:"code"`
	Name                string     `json:"name"`
	ContactName         string     `json:"contactName"`
	ContactEmail        string     `json:"contactEmail"`
	ContactPhone        string     `json:"contactPhone"`
	APIBaseURL          string     `json:"apiBaseUrl"`
	APIStatus           string     `json:"apiStatus"`
	Currency            string     `json:"currency"`
	Deposit             float64    `json:"deposit"`
	Balance             float64    `json:"balance"`
	Terms               string     `json:"terms"`
	Status              string     `json:"status"`
	LowBalanceThreshold float64    `json:"lowBalanceThreshold"`
	Countries           int        `json:"countries"`
	ESIMs               int        `json:"esims"`
	LastSyncedAt        *time.Time `json:"lastSyncedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type Tariff struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	SupplierID       string    `json:"supplierId"`
	SupplierName     string    `json:"supplierName"`
	ExternalID       string    `json:"externalId"`
	Name             string    `json:"name"`
	Country          string    `json:"country"`
	Region           string    `json:"region"`
	DataMB           int64     `json:"dataMb"`
	ValidityDays     int       `json:"validityDays"`
	Cost             float64   `json:"cost"`
	Price            float64   `json:"price"`
	MinPrice         float64   `json:"minPrice"`
	RecommendedPrice float64   `json:"recommendedPrice"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	Stock            int       `json:"stock"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ESIM struct {
	ID                 string     `json:"id"`
	Code               string     `json:"code"`
	SupplierID         string     `json:"supplierId"`
	SupplierName       string     `json:"supplierName"`
	TariffID           *string    `json:"tariffId,omitempty"`
	TariffName         string     `json:"tariffName"`
	Country            string     `json:"country"`
	ICCID              string     `json:"iccid"`
	IMSI               string     `json:"imsi"`
	ActivationCode     string     `json:"activationCode,omitempty"`
	SMDPAddress        string     `json:"smdpAddress,omitempty"`
	QRPayload          string     `json:"qrPayload,omitempty"`
	InstallURL         string     `json:"installUrl,omitempty"`
	IOSTapLink         string     `json:"iosTapLink,omitempty"`
	AndroidTapLink     string     `json:"androidTapLink,omitempty"`
	Status             string     `json:"status"`
	TrafficTotalMB     int64      `json:"trafficTotalMb"`
	TrafficRemainingMB int64      `json:"trafficRemainingMb"`
	CustomerName       string     `json:"customerName"`
	OrderID            *string    `json:"orderId,omitempty"`
	OrderNumber        *int64     `json:"orderNumber,omitempty"`
	ActivatedAt        *time.Time `json:"activatedAt,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type Order struct {
	ID                 string     `json:"id"`
	Number             int64      `json:"number"`
	CustomerID         string     `json:"customerId"`
	CustomerName       string     `json:"customerName"`
	CustomerPhone      string     `json:"customerPhone"`
	CustomerEmail      string     `json:"customerEmail"`
	TariffID           string     `json:"tariffId"`
	TariffName         string     `json:"tariffName"`
	SupplierID         string     `json:"supplierId"`
	SupplierName       string     `json:"supplierName"`
	ESIMID             *string    `json:"esimId,omitempty"`
	ICCID              string     `json:"iccid"`
	ActivationCode     string     `json:"activationCode,omitempty"`
	SMDPAddress        string     `json:"smdpAddress,omitempty"`
	QRPayload          string     `json:"qrPayload,omitempty"`
	InstallURL         string     `json:"installUrl,omitempty"`
	IOSTapLink         string     `json:"iosTapLink,omitempty"`
	AndroidTapLink     string     `json:"androidTapLink,omitempty"`
	OperatorID         *string    `json:"operatorId,omitempty"`
	OperatorName       string     `json:"operatorName"`
	Country            string     `json:"country"`
	PlanName           string     `json:"planName"`
	Cost               float64    `json:"cost"`
	Price              float64    `json:"price"`
	Fee                float64    `json:"fee"`
	Tax                float64    `json:"tax"`
	Profit             float64    `json:"profit"`
	Currency           string     `json:"currency"`
	PaymentMethod      string     `json:"paymentMethod"`
	PaymentStatus      string     `json:"paymentStatus"`
	Status             string     `json:"status"`
	ActivationAt       *time.Time `json:"activationAt,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	CancelledAt        *time.Time `json:"cancelledAt,omitempty"`
	CancellationReason string     `json:"cancellationReason"`
	ProviderOrderID    *string    `json:"providerOrderId,omitempty"`
	ProvisioningStatus string     `json:"provisioningStatus"`
	ProvisioningError  string     `json:"provisioningError,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}
