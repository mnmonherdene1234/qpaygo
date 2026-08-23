# QPay GO: Golang

Энэ нь qpay.mn-ийн ашиглалтад зориулагдсан бөгөөд Golang орчинд QPay v2 үйлчилгээг хэрэглэхэд тусална. Нэхэмжлэх, төлбөр, э-баримт зэрэг QPay-н v2 API-н бүх үндсэн боломжийг дэмжинэ.

## Татаж

```bash
go get github.com/mnmonherdene1234/qpaygo
```

## Шинэ QPay Client үүсгэх

Шинэ QPay Client үүсгэх (token-г шууд авна):

```go
ctx := context.Background()
client, err := qpaygo.NewQPayClient(ctx, "USERNAME", "PASSWORD", "INVOICE_CODE")
```

- "USERNAME": Нэвтрэх нэр
- "PASSWORD": Нууц үг
- "INVOICE_CODE": Нэхэмжлэлийн код

## Token авахгүй үүсгэх (lazy)

Token-ийг шууд авахгүйгээр client үүсгэнэ. Эхний хүсэлт илгээх үед автоматаар token авна:

```go
client := qpaygo.NewQPayClientLazy("USERNAME", "PASSWORD", "INVOICE_CODE")
```

## Sandbox орчинд ашиглах

Үйлдвэрлэлийн хост нь `qpaygo.DefaultHost`. Sandbox орчинд туршихдаа `WithHost` ашиглана:

```go
client, err := qpaygo.NewQPayClient(ctx, "USERNAME", "PASSWORD", "INVOICE_CODE",
	qpaygo.WithHost(qpaygo.SandboxHost),
)
```

Мөн `WithHTTPClient` болон `WithTimeout` бэлэн байна.

## Token сэргээх

`CheckTokenAndRefresh` (бүх типизацлагдсан метод дотор автоматаар дуудагддаг) нь token дуусахад эхлээд `POST /v2/auth/refresh` ашиглан сэргээхийг оролдоно, амжилтгүй бол л дахин бүтэн нэвтрэлт хийнэ — та өөрөө дуудах шаардлагагүй.

## Нэхэмжлэх үүсгэх (энгийн)

```go
response, err := client.CreateSimpleInvoice(ctx, qpaygo.CreateSimpleInvoiceRequest{
	SenderInvoiceNo:     "senderInvoiceNo35",            // Илгээгчийн дугаар
	InvoiceReceiverCode: "invoiceReceiverCode99",         // Хүлээн авагчийн код
	InvoiceDescription:  "Нэхэмжлэлийн гүйлгээний утга", // Гүйлгээний тайлбар
	Amount:              2000,                            // Гүйлгээний дүн
	CallbackURL:         "https://example.com/callback",  // Callback URL
})
```

## Нэхэмжлэх үүсгэх (бүрэн)

Мөр (`lines`), хөнгөлөлт/нэмэлт төлбөр/татвар, болон дэд дансдад хуваан тооцох (`transactions`) зэргийг дэмжинэ:

```go
response, err := client.CreateInvoice(ctx, qpaygo.CreateInvoiceRequest{
	SenderInvoiceNo:     "senderInvoiceNo35",
	InvoiceReceiverCode: "invoiceReceiverCode99",
	InvoiceDescription:  "Нэхэмжлэлийн гүйлгээний утга",
	CallbackURL:         "https://example.com/callback",
	Lines: []qpaygo.Line{
		{
			LineDescription: "Бараа 1",
			LineQuantity:    1,
			LineUnitPrice:   2000,
			Taxes: []qpaygo.LineTax{
				{TaxCode: "VAT", Description: "НӨАТ", Amount: 180},
			},
		},
	},
})
```

## Нэхэмжлэх авах

```go
response, err := client.GetInvoice(ctx, "0c32b23f-f162-4caf-94dd-09a49952a9ba")
```

## Нэхэмжлэх цуцлах

```go
err := client.CancelInvoice(ctx, "0c32b23f-f162-4caf-94dd-09a49952a9ba")
```

## Төлбөрийн мэдээлэл авах

> **АНХААРУУЛГА!** Cron job ашиглан төлбөр байнга шалгахыг QPay хориглодог. Зөвхөн callback URL хүсэлт ирсний дараа шалгана уу.
>
> Callback хүсэлт нь өөрөө төлбөр амжилттай болсны нотолгоо биш (QPay v2 API-д гарын үсэг баталгаажуулалт байхгүй) — үргэлж `GetPayment`/`CheckPayment`-ээр дахин баталгаажуулна уу.

```go
payment, err := client.GetPayment(ctx, "493622150113497")
```

## Төлбөр шалгах

`Offset` сонголттой — `nil` үлдээвэл хүсэлтэд огт илгээгдэхгүй (QPay зөвшөөрдөг). Явуулахдаа `page_number`/`page_limit`-ийг **[1, 100]** дотор заавал өгнө — 0 утгатай offset-г QPay `MIN_NUMBER` алдаагаар татгалзана.

```go
result, err := client.CheckPayment(ctx, qpaygo.CheckPaymentRequest{
	ObjectType: qpaygo.ObjectTypeInvoice,
	ObjectID:   "0c32b23f-f162-4caf-94dd-09a49952a9ba",
	Offset:     &qpaygo.Offset{PageNumber: 1, PageLimit: 100},
})
```

## Төлбөр цуцлах / буцаах

> **АНХААРУУЛГА!** Зөвхөн картын гүйлгээнд ажиллана. P2P (банкны шилжүүлэг) төлбөрт хэрэглэвэл `PAYMENT_SETTLED` алдаа буцна.

```go
err := client.CancelPayment(ctx, "493622150113497", qpaygo.CancelPaymentRequest{Note: "Захиалга цуцлагдсан"})
err  = client.RefundPayment(ctx, "493622150113497", qpaygo.RefundPaymentRequest{Note: "Буцаалт"})
```

## Төлбөрийн жагсаалт авах

`Offset`-ийг `nil` орхивол `{1, 100}` гэж автоматаар бөглөнө (энэ endpoint-д QPay offset заавал шаарддаг).

> **АНХААРУУЛГА:** `ObjectTypeMerchant`-ийн `object_id` нь access token-ийн JWT доторх **`merchant_id` (UUID)** байна — invoice code БИШ. Invoice code явуулбал `401 PERMISSION_DENIED` буцна. `ObjectTypeInvoice`-ийн `object_id` нь харин invoice code.

```go
list, err := client.ListPayments(ctx, qpaygo.ListPaymentsRequest{
	ObjectType: qpaygo.ObjectTypeMerchant,
	ObjectID:   "1c46a9ec-3045-4341-adb9-ffe6feb9d0df", // merchant UUID (JWT-ийн merchant_id claim)
	StartDate:  qpaygo.FormatQPayTime(time.Now().AddDate(0, 0, -7)),
	EndDate:    qpaygo.FormatQPayTime(time.Now()),
	Offset:     &qpaygo.Offset{PageNumber: 1, PageLimit: 100},
})
```

## И-баримт

```go
ebarimt, err := client.CreateEbarimt(ctx, qpaygo.CreateEbarimtRequest{
	PaymentID:           "493622150113497",
	EbarimtReceiverType: qpaygo.EbarimtReceiverCitizen,
})

ebarimt, err = client.GetEbarimt(ctx, ebarimt.ID)

// АНХААРУУЛГА: зарим худалдагчийн тохиргоонд QPay `EBARIMT_CANCEL_NOTSUPPERDED`
// алдаа буцааж болно — энэ бол баримтжуулагдсан, хүлээгдэж буй хариу.
err = client.CancelEbarimt(ctx, ebarimt.ID)
```

## Callback URL зохицуулах

```go
http.HandleFunc("/qpay/callback", func(w http.ResponseWriter, r *http.Request) {
	payment, err := client.VerifyCallback(r.Context(), r)
	if err != nil {
		log.Println("qpay callback verify failed:", err)
		qpaygo.WriteCallbackAck(w) // QPay нэвтэрсэн гэдгийг мэдэгдэхийн тулд эсэн мэт ч ACK хийнэ
		return
	}

	if payment.PaymentStatus == qpaygo.PaymentStatusPaid {
		// захиалгыг гүйцэтгэх
	}

	qpaygo.WriteCallbackAck(w)
})
```

`ExtractPaymentID`/`WriteCallbackAck` тусад нь ч ашиглаж болно.

## Алдаа зохицуулах

QPay-ийн бүтэцлэгдсэн алдааг `*qpaygo.APIError`-оор шалгаж болно:

```go
_, err := client.CreateSimpleInvoice(ctx, req)
var apiErr *qpaygo.APIError
if errors.As(err, &apiErr) {
	switch apiErr.Code {
	case qpaygo.ErrInvoiceCodeInvalid:
		// ...
	case qpaygo.ErrPaymentSettled:
		// картын бус гүйлгээг цуцлах/буцаах гэж оролдсон
	case qpaygo.ErrNoCredentialsProduction:
		// access token хүчингүй/хугацаа дууссан — шинэ клиент үүсгэх
		// (production-ийн бичиглэл; sandbox нь ErrNoCredentials илгээдэг)
	case qpaygo.ErrSystemBusy:
		// QPay талын түр зуурын алдаа (500)
	}
}
```

## Хүсэлт илгээх

Өөрийн хэрэгцээнд тохирсон API хандалтыг илгээж болно. Анхаарах гол зүйл бол хүсэлтийн header хэсэгт token мэдээллийг автоматаар оруулсан байдаг:

```go
response, err := client.Request(ctx, http.MethodGet, "/v2/invoice/"+invoiceID, nil)
if err != nil {
	return err
}
defer response.Body.Close() // Request танд raw response өгдөг — body-г өөрөө хаах ёстой
```

## Тест ажиллуулах

Сүлжээгүй, mock хийсэн unit тестүүд (нэвтрэх мэдээлэл шаардахгүй):

```bash
go test ./...
```

Жинхэнэ QPay API-тай холбогдох интеграцийн тестүүд (`-tags=integration`), анхдагчаар sandbox хост руу орно:

```bash
QPAY_USERNAME=... QPAY_PASSWORD=... QPAY_INVOICE_CODE=... go test -tags=integration ./...
```

Үйлдвэрлэлийн орчинд турших бол `QPAY_HOST=https://merchant.qpay.mn` нэмж өгнө. Нэвтрэх мэдээллийг хэзээ ч репод commit хийхгүй — зөвхөн орчны хувьсагчаар дамжуулна.
