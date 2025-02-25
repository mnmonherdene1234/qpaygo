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
	InvoiceCode         string  `json:"invoice_code"`          // QPay-ээс өгсөн нэхэмжлэхийн код
	SenderInvoiceNo     string  `json:"sender_invoice_no"`     // Байгууллагаас үүсгэх давтагдашгүй нэхэмжлэлийн дугаар
	InvoiceReceiverCode string  `json:"invoice_receiver_code"` // Байгууллагын нэхэмжлэхийг хүлээн авч буй харилцагчийн дахин давтагдашгүй дугаар
	InvoiceDescription  string  `json:"invoice_description"`   // Нэхэмжлэлийн утга, гүйлгээний утга
	Amount              float64 `json:"amount"`                // Мөнгөн дүн
	CallbackURL         string  `json:"callback_url"`          // Төлбөр амжилттай төлөгдсөн тохиолдолд дуудагдах URL
}
