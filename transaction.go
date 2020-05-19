package api

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
)

type createdTransaction struct {
	TransactionId        int    `json:"rcTransactionId"`
	ConfirmationRequired bool   `json:"confirmationRequired"`
	ConfirmationText     string `json:"confirmationText"`
	ObjectType           string `json:"objType"`
	ObjectId             string `json:"objId"`
	Type                 string `json:"type"`
	Data                 string `json:"data"`
}

type transactionCreateRequest struct {
	request
	ObjectType string `json:"objType"`
	ObjectId   string `json:"objId"`
	Type       string `json:"type"`
	Confirm    bool   `json:"confirmation"`
	Data       string `json:"data"`
}

type transactionCreateResponse struct {
	response
	Transaction *createdTransaction `json:"rcTransaction"`
}

type existingTransaction struct {
	TransactionId string `json:"rcTransactionId"`
	ObjectType    string `json:"objType"`
	ObjectId      string `json:"objId"`
	Type          string `json:"method"`
	Data          string `json:"data"`
	Status        string `json:"status"`
	StatusInfo    string `json:"statusInfo"`
}

type transactionGetRequest struct {
	request
	TransactionId int `json:"rcTransactionId"`
}

type transactionGetResponse struct {
	response
	Transaction *existingTransaction `json:"rcTransaction"`
}

type TransactionFilter struct {
	ObjectType   string   `json:"objType"`
	ObjectId     []string `json:"objId,omitempty"`
	Type         []string `json:"method,omitempty"`
	Status       []string `json:"status,omitempty"`
	CustomerId   []string `json:"customerId,omitempty"`
	ResultStart  int      `json:"resStart,omitempty"`
	ResultWindow int      `json:"resWindow,omitempty"`
}

type transactionGetAllRequest struct {
	request
	TransactionFilter
}

type transactionGetAllResponse struct {
	response
	Matches      int                   `json:"matches"`
	Transactions []existingTransaction `json:"rcTransactions"`
}

type Transaction struct {
	TransactionId        string
	ObjectType           string
	ObjectId             string
	Type                 string
	Data                 string
	ConfirmationRequired bool
	ConfirmationText     string
	Status               string
	StatusInfo           string
}

type TransactionStartupCloudInit struct {
	MetaData      string `json:"metaData,omitempty"`
	NetworkConfig string `json:"networkConfig,omitempty"`
	UserData      string `json:"userData,omitempty"`
}

type TransactionStartupData struct {
	DeployMediaImageAccessKey    string `json:"deployMediaImageAccessKey,omitempty"`
	DeployMediaImageAccessSecret string `json:"deployMediaImageAccessSecret,omitempty"`
	DeployMediaImageBucket       string `json:"deployMediaImageBucket,omitempty"`
	DeployMediaImageId           string `json:"deployMediaImageId,omitempty"`
	DeployMediaImagePath         string `json:"deployMediaImagePath,omitempty"`

	CloudInit TransactionStartupCloudInit `json:"cloudInit"`
}

const (
	TransactionObjectTypeDevice = "DEVICE"

	TransactionStatusCommenced = "COMMENCED"
	TransactionStatusCompleted = "COMPLETED"
	TransactionStatusPending   = "PENDING"

	TransactionTypeCancel        = "CANCEL"
	TransactionTypeCloseVNC      = "CLOSEVNC"
	TransactionTypeForceShutdown = "FORCESHUTDOWN"
	TransactionTypeOpenVNC       = "OPENVNC" // data parameter contains public IP that allows VNC
	TransactionTypeRefreshConfig = "REFRESHCONFIG"
	TransactionTypeSafeShutdown  = "SAFESHUTDOWN"
	TransactionTypeShutdown      = "SHUTDOWN"
	TransactionTypeStartup       = "STARTUP"
)

func (t *createdTransaction) ToTransaction() *Transaction {
	return &Transaction{
		TransactionId:        strconv.Itoa(t.TransactionId),
		ObjectType:           t.ObjectType,
		ObjectId:             t.ObjectId,
		Type:                 t.Type,
		Data:                 t.Data,
		ConfirmationRequired: t.ConfirmationRequired,
		ConfirmationText:     t.ConfirmationText,
	}
}

func (t *existingTransaction) ToTransaction() *Transaction {
	return &Transaction{
		TransactionId: t.TransactionId,
		ObjectType:    t.ObjectType,
		ObjectId:      t.ObjectId,
		Type:          t.Type,
		Data:          t.Data,
		Status:        t.Status,
		StatusInfo:    t.StatusInfo,
	}
}

func (c *client) TransactionDeviceStartup(deviceId string, data TransactionStartupData) (*Transaction, error) {
	var encodedData, err = json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to JSON encode transaction startup data")
	}

	return c.transactionCreateInternal(
		TransactionTypeStartup,
		TransactionObjectTypeDevice,
		deviceId,
		true,
		string(encodedData),
	)
}

func (c *client) TransactionCreate(transactionType string, objectType string, objectId string, confirm bool) (*Transaction, error) {
	return c.transactionCreateInternal(
		transactionType,
		objectType,
		objectId,
		confirm,
		"",
	)
}

func (c *client) transactionCreateInternal(transactionType string, objectType string, objectId string, confirm bool, data string) (*Transaction, error) {
	if transactionType == "" {
		return nil, errors.New("transactionType parameter is required.")
	}

	if objectType == "" {
		return nil, errors.New("objectType parameter is required.")
	}

	if objectId == "" {
		return nil, errors.New("objectId parameter is required.")
	}

	req := &transactionCreateRequest{
		request:    c.newRequest("rctransaction.create"),
		Type:       transactionType,
		ObjectType: objectType,
		ObjectId:   objectId,
		Confirm:    confirm,
		Data:       data,
	}

	var resp transactionCreateResponse
	err := c.httpPostJson(req, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "TransactionCreate request failed.")
	}

	if resp.Code != "OK" || resp.Transaction == nil {
		return nil, newApiError(resp.response, nil)
	}

	return resp.Transaction.ToTransaction(), nil
}

func (c *client) TransactionGet(transactionId string) (*Transaction, error) {
	numericId, err := strconv.Atoi(transactionId)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid transactionId")
	}

	req := &transactionGetRequest{
		request:       c.newRequest("rctransaction.get"),
		TransactionId: numericId,
	}

	var resp transactionGetResponse
	err = c.httpPostJson(req, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "TransactionGet request failed.")
	}

	if resp.Code != "OK" || resp.Transaction == nil {
		return nil, newApiError(resp.response, nil)
	}

	return resp.Transaction.ToTransaction(), nil
}

func (c *client) TransactionGetAll(filter TransactionFilter) ([]Transaction, int, error) {
	if filter.ObjectType == "" {
		return nil, 0, errors.New("ObjectType field of TransactionFilter is required.")
	}

	req := &transactionGetAllRequest{
		request:           c.newRequest("rctransaction.getall"),
		TransactionFilter: filter,
	}

	var resp transactionGetAllResponse
	err := c.httpPostJson(req, &resp)
	if err != nil {
		return nil, 0, errors.Wrap(err, "TransactionGetAll request failed.")
	}

	if resp.Code != "OK" || resp.Transactions == nil {
		return nil, 0, newApiError(resp.response, nil)
	}

	transactions := make([]Transaction, len(resp.Transactions))
	for index, transaction := range resp.Transactions {
		transactions[index] = *transaction.ToTransaction()
	}

	return transactions, resp.Matches, nil
}
