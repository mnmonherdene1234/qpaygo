# QPay GO: Golang

Энэ нь qpay.mn-ийн ашиглалтад зориулагдсан бөгөөд Golang орчинд QPay үйлчилгээг хэрэглэхэд тусална.

## Шинэ QPay Client үүсгэх

Шинэ QPay Client үүсгэх:

```go
client, err := NewQPayClient("USERNAME", "PASSWORD", "INVOICE_CODE")
```

- "USERNAME": Нэвтрэх нэр
- "PASSWORD": Нууц үг
- "INVOICE_CODE": Нэхэмжлэлийн код

## Нэхэмжлэх үүсгэх

Энд нэхэмжлэх үүсгэх үйлдлийн жишээг харуулж байна:

```go
response, err := client.CreateAmountInvoice(
    "senderInvoiceNo35",          // Илгээгчийн дугаар
    "invoiceReceiverCode99",      // Хүлээн авагчийн код
    "Нэхэмжлэлийн гүйлгээний утга", // Гүйлгээний тайлбар
    2000,                         // Гүйлгээний дүн
    "https://example.com/callback", // Callback URL
)
```

## Нэхэмжлэх авах

Нэхэмжлэхийн мэдээллийг авах:

```go
response, err := client.GetInvoice("0c32b23f-f162-4caf-94dd-09a49952a9ba")
```

## Хүсэлт илгээх

Өөрийн хэрэгцээнд тохирсон API хандалтыг илгээж болно. Анхаарах гол зүйл бол хүсэлтийн header хэсэгт token мэдээллийг оруулсан байдаг:

```go
response, err := q.Request(http.MethodGet, "/v2/invoice/"+invoiceID, nil)
```
