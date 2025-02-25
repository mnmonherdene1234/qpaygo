package qpaygo

type QPayTokenResponse struct {
	TokenType        string `json:"token_type"`         // Токены төрөл
	RefreshExpiresIn int64  `json:"refresh_expires_in"` // Шинэчлэх хугацаа дуусах хугацаа (секундээр)
	RefreshToken     string `json:"refresh_token"`      // Шинэчлэх токен
	AccessToken      string `json:"access_token"`       // Нэвтрэх токен
	ExpiresIn        int64  `json:"expires_in"`         // Хугацаа дуусах хугацаа (секундээр)
	Scope            string `json:"scope"`              // Хүрээ
	NotBeforePolicy  string `json:"not-before-policy"`  // Бодлого эхлэх хугацаа
	SessionState     string `json:"session_state"`      // Сессийн төлөв
}
