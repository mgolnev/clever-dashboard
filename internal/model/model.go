// Package model содержит нейтральные доменные типы, общие для нескольких
// сервисов (ingestion, orders, metrics). Не зависит ни от одного домена.
package model

import "time"

// Order — нормализованный заказ из выгрузки Битрикса.
type Order struct {
	OrderNumber     string      `json:"orderNumber"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	Customer        string      `json:"customer"`
	Email           string      `json:"email"`
	Phone           string      `json:"phone"`
	TotalAmount     int         `json:"totalAmount"`
	DeliveryCost    int         `json:"deliveryCost"`
	StatusRaw       string      `json:"statusRaw"`
	StatusStage     string      `json:"statusStage"`
	IsPaid          bool        `json:"isPaid"`
	IsCanceled      bool        `json:"isCanceled"`
	PaymentSystem   string      `json:"paymentSystem"`
	DeliveryService string      `json:"deliveryService"`
	Channel         string      `json:"channel"`
	Region          string      `json:"region"`
	City            string      `json:"city"`
	LocationRaw     string      `json:"locationRaw"`
	HasProblem      bool        `json:"hasProblem"`
	ProblemDesc     string      `json:"problemDesc"`
	CancelReason    string      `json:"cancelReason"`
	Items           []OrderItem `json:"items"`
}

// OrderItem — позиция заказа (распарсена из столбца «Позиции»).
type OrderItem struct {
	OfferID  string `json:"offerId"`
	Name     string `json:"name"`
	Qty      int    `json:"qty"`
	Price    int    `json:"price"`
	LineSum  int    `json:"lineSum"`
	Brand    string `json:"brand"`
	Category string `json:"category"`
	Gender   string `json:"gender"`
	Size     string `json:"size"`
}

// ImportResult — итог загрузки файла.
type ImportResult struct {
	ImportID       int64      `json:"importId"`
	Filename       string     `json:"filename"`
	RowsTotal      int        `json:"rowsTotal"`
	OrdersImported int        `json:"ordersImported"`
	ItemsImported  int        `json:"itemsImported"`
	PeriodStart    *time.Time `json:"periodStart"`
	PeriodEnd      *time.Time `json:"periodEnd"`
}
