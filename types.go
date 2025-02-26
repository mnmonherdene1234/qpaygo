package qpaygo

type TokenResponse struct {
	TokenType        string `json:"token_type"`         // Токены төрөл
	RefreshExpiresIn int64  `json:"refresh_expires_in"` // Шинэчлэх хугацаа дуусах хугацаа (секундээр)
	RefreshToken     string `json:"refresh_token"`      // Шинэчлэх токен
	AccessToken      string `json:"access_token"`       // Нэвтрэх токен
	ExpiresIn        int64  `json:"expires_in"`         // Хугацаа дуусах хугацаа (секундээр)
	Scope            string `json:"scope"`              // Хүрээ
	NotBeforePolicy  string `json:"not-before-policy"`  // Бодлого эхлэх хугацаа
	SessionState     string `json:"session_state"`      // Сессийн төлөв
}

type CreateAmountInvoiceRequest struct {
	InvoiceCode         string `json:"invoice_code"`          // QPay-ээс өгсөн нэхэмжлэхийн код
	SenderInvoiceNo     string `json:"sender_invoice_no"`     // Байгууллагаас үүсгэх давтагдашгүй нэхэмжлэлийн дугаар
	InvoiceReceiverCode string `json:"invoice_receiver_code"` // Байгууллагын нэхэмжлэхийг хүлээн авч буй харилцагчийн дахин давтагдашгүй дугаар
	InvoiceDescription  string `json:"invoice_description"`   // Нэхэмжлэлийн утга, гүйлгээний утга
	Amount              uint   `json:"amount"`                // Мөнгөн дүн
	CallbackURL         string `json:"callback_url"`          // Төлбөр амжилттай төлөгдсөн тохиолдолд дуудагдах URL
}

type URL struct {
	Name        string `json:"name"`        // нэр
	Description string `json:"description"` // тайлбар
	Logo        string `json:"logo"`        // лого
	Link        string `json:"link"`        // хаяг
}

type CreateAmountInvoiceResponse struct {
	InvoiceID    string `json:"invoice_id"`    // Нэхэмжлэлийн дугаар
	QRText       string `json:"qr_text"`       // QR кодны текст
	QRImage      string `json:"qr_image"`      // QR кодны зураг
	QPayShortURL string `json:"qPay_shortUrl"` // QPay-н товчлол
	URLs         []URL  `json:"urls"`          // Линкүүд
}

type Line struct {
	TaxProductCode  string `json:"tax_product_code"` // Татварын бүтээгдэхүүний код
	LineDescription string `json:"line_description"` // Мөрийн тайлбар
	LineQuantity    string `json:"line_quantity"`    // Мөрийн тоо хэмжээ
	LineUnitPrice   string `json:"line_unit_price"`  // Мөрийн нэгж үнэ
	Note            string `json:"note"`             // Тэмдэглэл
	Discounts       []any  `json:"discounts"`        // Хөнгөлөлтүүд
	Surcharges      []any  `json:"surcharges"`       // Нэмэлт төлбөрүүд
	Taxes           []any  `json:"taxes"`            // Татварууд
}

type GetInvoiceResponse struct {
	InvoiceID          string  `json:"invoice_id"`           // Нэхэмжлэлийн дугаар
	InvoiceStatus      string  `json:"invoice_status"`       // Нэхэмжлэлийн төлөв ("OPEN" | "CLOSED")
	SenderInvoiceNo    string  `json:"sender_invoice_no"`    // Байгууллагаас үүсгэх давтагдашгүй нэхэмжлэлийн дугаар
	SenderBranchCode   string  `json:"sender_branch_code"`   // Байгууллагын салбарын код
	SenderBranchData   string  `json:"sender_branch_data"`   // Байгууллагын салбарын мэдээлэл
	SenderStaffCode    string  `json:"sender_staff_code"`    // Байгууллагын ажилтны код
	SenderStaffData    string  `json:"sender_staff_data"`    // Байгууллагын ажилтны мэдээлэл
	SenderTerminalCode string  `json:"sender_terminal_code"` // Байгууллагын терминалын код
	SenderTerminalData string  `json:"sender_terminal_data"` // Байгууллагын терминалын мэдээлэл
	InvoiceDescription string  `json:"invoice_description"`  // Нэхэмжлэлийн утга, гүйлгээний утга
	InvoiceDueDate     string  `json:"invoice_due_date"`     // Нэхэмжлэлийн дуусах хугацаа
	EnableExpiry       bool    `json:"enable_expiry"`        // Дуусах хугацааг идэвхжүүлэх эсэх
	ExpiryDate         string  `json:"expiry_date"`          // Дуусах хугацаа
	AllowPartial       bool    `json:"allow_partial"`        // Хэсэгчлэн төлөхийг зөвшөөрөх эсэх
	MinimumAmount      float64 `json:"minimum_amount"`       // Хамгийн бага дүн
	AllowExceed        bool    `json:"allow_exceed"`         // Хэтрүүлэхийг зөвшөөрөх эсэх
	MaximumAmount      string  `json:"maximum_amount"`       // Хамгийн их дүн
	TotalAmount        string  `json:"total_amount"`         // Нийт дүн
	GrossAmount        uint    `json:"gross_amount"`         // Нийт дүн
	TaxAmount          uint    `json:"tax_amount"`           // Татварын дүн
	SurchargeAmount    uint    `json:"surcharge_amount"`     // Нэмэлт төлбөрийн дүн
	DiscountAmount     uint    `json:"discount_amount"`      // Хөнгөлөлтийн дүн
	CallbackURL        string  `json:"callback_url"`         // Төлбөр амжилттай төлөгдсөн тохиолдолд дуудагдах URL
	Note               string  `json:"note"`                 // Тэмдэглэл
	Lines              []Line  `json:"lines"`                // Мөрүүд
	Transactions       []any   `json:"transactions"`         // Гүйлгээ
	Inputs             []any   `json:"inputs"`               // Оролтууд
}
