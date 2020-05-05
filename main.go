package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"unicode/utf8"

	"github.com/gocarina/gocsv"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func main() {
	// Shopifyの注文データは最大50件
	orders, err := ImportShopifyOrders("shopify-orders.csv")
	if err != nil {
		panic(err)
	}
	// クリックポストにアップロードできる送り状ラベルは最大40件まで
	const maxClickpostShippingLabels = 40
	for i, chunkedOrders := range ChunkShopifyOrders(orders, maxClickpostShippingLabels) {
		if err := ExportClickpostShippingLabels(fmt.Sprintf("clickpost-shipping-labels-%d.csv", i), chunkedOrders); err != nil {
			panic(err)
		}
	}
}

// ImportShopifyOrders Shopifyの注文データをCSVとしてインポート
func ImportShopifyOrders(filename string) ([]*ShopifyOrder, error) {
	inFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	var orders []*ShopifyOrder
	if err := gocsv.UnmarshalFile(inFile, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func ChunkShopifyOrders(items []*ShopifyOrder, chunkSize int) (chunks [][]*ShopifyOrder) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

// ExportClickpostShippingLabels Shopifyの注文データをクリックポストの送り状発行用CSVに変換してエクスポート
func ExportClickpostShippingLabels(filename string, orders []*ShopifyOrder) error {
	var shippingLabels []*ClickpostShippingLabel
	for _, o := range orders {
		label := o.ToClickpostShippingLabel()
		if err := label.Validate(); err != nil {
			log.Printf("注文番号:%s エラー:%v\n", o.Name, err)
			continue
		}
		shippingLabels = append(shippingLabels, label)
	}
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()
	gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
		writer := csv.NewWriter(transform.NewWriter(out, japanese.ShiftJIS.NewEncoder()))
		writer.UseCRLF = true
		return gocsv.NewSafeCSVWriter(writer)
	})
	if err := gocsv.MarshalFile(&shippingLabels, outFile); err != nil {
		return err
	}
	return nil
}

type ShopifyOrder struct {
	Name             string `csv:"Name"`              // ストア管理画面に表示される注文番号
	ShippingName     string `csv:"Shipping Name"`     // お客様の氏名
	ShippingStreet   string `csv:"Shipping Street"`   // 配送先住所として入力されている町名
	ShippingAddress1 string `csv:"Shipping Address1"` // 150 Elginなど配送先住所の1行目
	ShippingAddress2 string `csv:"Shipping Address2"` // Suite 800など配送先住所の2行目。この欄は空欄の場合があります
	ShippingCity     string `csv:"Shipping City"`     // 配送先住所の都市
	ShippingZip      string `csv:"Shipping Zip"`      // 配送先住所の郵便番号
	ShippingProvince string `csv:"Shipping Province"` // 配送先の都道府県
}

func (s ShopifyOrder) ToClickpostShippingLabel() *ClickpostShippingLabel {
	return &ClickpostShippingLabel{
		ShippingZip:       s.ShippingZip,
		ShippingName:      s.ShippingName,
		ShippingNameTitle: "様",
		ShippingAddress1:  s.ShippingProvince + s.ShippingCity,
		ShippingAddress2:  s.ShippingStreet + s.ShippingAddress1,
		ShippingAddress3:  s.ShippingAddress2,
		ShippingContents:  "サプリメント",
	}
}

type ClickpostShippingLabel struct {
	ShippingZip       string `csv:"お届け先郵便番号"`  // お届け先郵便番号
	ShippingName      string `csv:"お届け先氏名"`    // お届け先氏名
	ShippingNameTitle string `csv:"お届け先敬称"`    // お届け先敬称
	ShippingAddress1  string `csv:"お届け先住所1行目"` // お届け先住所1行目
	ShippingAddress2  string `csv:"お届け先住所2行目"` // お届け先住所2行目
	ShippingAddress3  string `csv:"お届け先住所3行目"` // お届け先住所3行目
	ShippingAddress4  string `csv:"お届け先住所4行目"` // お届け先住所4行目
	ShippingContents  string `csv:"内容品"`       // 内容品
}

// Validate ...
func (c ClickpostShippingLabel) Validate() error {
	if c.ShippingZip == "" {
		return errors.New("お届け先郵便番号は必須です")
	}
	if c.ShippingName == "" {
		return errors.New("お届け先氏名は必須です")
	}
	if utf8.RuneCountInString(c.ShippingName) > 20 {
		return errors.New("お届け先氏名は全角20文字までです")
	}
	if c.ShippingAddress1 == "" {
		return errors.New("お届け先住所1行目は必須です")
	}
	if utf8.RuneCountInString(c.ShippingAddress1) > 20 {
		return errors.New("お届け先住所1行目は全角20文字までです")
	}
	if c.ShippingAddress2 == "" {
		return errors.New("お届け先住所2行目は必須です")
	}
	if utf8.RuneCountInString(c.ShippingAddress2) > 20 {
		return errors.New("お届け先住所2行目は全角20文字までです")
	}
	if utf8.RuneCountInString(c.ShippingAddress3) > 20 {
		return errors.New("お届け先住所3行目は全角20文字までです")
	}
	if utf8.RuneCountInString(c.ShippingAddress4) > 20 {
		return errors.New("お届け先住所4行目は全角20文字までです")
	}
	if utf8.RuneCountInString(c.ShippingContents) > 15 {
		return errors.New("内容品は全角15文字までです")
	}
	return nil
}
