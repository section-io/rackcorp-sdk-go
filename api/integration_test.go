package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

type cloudInit struct {
	UserData string `json:"userData"`
	Metadata string `json:"metaData"`
}

type startupData struct {
	CloudInit          cloudInit `json:"cloudInit"`
	DeployMediaImageId string    `json:"deployMediaImageId"`
}

func connectSSH(t *testing.T, host, user, password string) {
	t.Helper()
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:22", host),
		config,
	)
	require.Nil(t, err, "ssh dial")

	session, err := client.NewSession()
	require.Nil(t, err, "ssh newsession")
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	err = session.Run("/usr/bin/whoami")
	require.Nil(t, err, "ssh run")
	fmt.Println(b.String())
}

func waitForPendingTransactions(t *testing.T, client Client, deviceId string) {
	t.Helper()
	filter := TransactionFilter{
		ObjectType:   TransactionObjectTypeDevice,
		ObjectId:     []string{deviceId},
		Status:       []string{TransactionStatusPending, TransactionStatusCommenced},
		ResultWindow: 1,
	}
	for {
		transactions, matches, err := client.TransactionGetAll(filter)
		require.Nil(t, err, "TransactionGetAll failed")
		if matches == 0 {
			break
		}
		t.Logf("pending transactions: %#v", transactions)
		time.Sleep(2 * time.Second)
	}
}

func TestCloudInit(t *testing.T) {
	client, err := NewClientFromEnvironment()
	if err != nil {
		t.Skipf("Skipping... configure the required environment variables to execute. %s", err)
	}

	customerId := os.Getenv("RACKCORP_API_TEST_CUSTOMER_ID")
	if customerId == "" {
		t.Skipf("Skipping... configure the RACKCORP_API_TEST_CUSTOMER_ID environment variable to execute.")
	}

	rootPassword := "df6kjgdf" // TODO random
	credentials := []Credential{
		{
			Username: "root",
			Password: rootPassword,
		},
	}
	require.NotNil(t, credentials, "creds")

	productDetails := ProductDetails{
		//Credentials:      credentials,
		CpuCount:         1,
		Hostname:         "rackcorp-sdk-go-api-001", // TODO random
		MemoryGB:         1,
		Storage:          []Storage{
			{
				SizeGB:      40,
				StorageType: StorageTypeMagnetic,
			},
		},
		FirewallPolicies: []FirewallPolicy{},
	}

	productCode := GetVirtualServerProductCode("PERFORMANCE", "AU")
	createdOrder, err := client.OrderCreate(
		productCode,
		customerId,
		productDetails,
	)
	require.Nil(t, err, "OrderCreate failed")

	orderId := createdOrder.OrderId
	confirmedOrder, err := client.OrderConfirm(orderId)
	require.Nil(t, err, "OrderConfirm failed")

	contractCount := len(confirmedOrder.ContractIds)
	require.Equal(t, 1, contractCount, "Expected only one contract")

	contractId := confirmedOrder.ContractIds[0]
	require.NotEqual(t, "", contractId)

	var contract *OrderContract
	for {
		contract, err = client.OrderContractGet(contractId)
		require.Nil(t, err, "OrderContractGet failed")
		if contract.Status == "ACTIVE" {
			break
		}
		require.Equal(t, "PENDING", contract.Status)
		time.Sleep(2 * time.Second)
	}

	deviceId := contract.DeviceId
	require.NotEqual(t, "", contract.DeviceId, "Expected a device ID")
	assert.Equal(t, "ACTIVE", contract.Status, "contract status")

	device, err := client.DeviceGet(deviceId)
	require.Nil(t, err, "DeviceGet failed")

	waitForPendingTransactions(t, client, deviceId)

	createdTransaction, err := client.TransactionCreate(
		TransactionTypeRefreshConfig,
		TransactionObjectTypeDevice,
		deviceId,
		true,
	)
	require.Nil(t, err, "TransactionCreate 'refreshconfig' failed")
	require.NotEqual(t, "", createdTransaction.TransactionId, "Expected a transaction ID")

	waitForPendingTransactions(t, client, deviceId)

	data, err := json.Marshal(startupData{
		CloudInit: cloudInit{
			UserData: `#cloud-config
ssh_pwauth: true
users:
  - name: ubuntu
    shell: /bin/bash
    lock_passwd: false
    plain_test_passwd: ` + rootPassword,
			Metadata: "",
		},
		DeployMediaImageId: "173",
	})
	require.Nil(t, err, "json.Marshal failed")

	createdTransaction, err = client.TransactionCreateWithData(
		TransactionTypeStartup,
		TransactionObjectTypeDevice,
		deviceId,
		true,
		string(data),
	)
	require.Nil(t, err, "TransactionCreate 'startup' failed")
	require.NotEqual(t, "", createdTransaction.TransactionId, "Expected a transaction ID")

	waitForPendingTransactions(t, client, deviceId)

	connectSSH(t, device.PrimaryIP, "ubuntu", rootPassword)

	createdTransaction, err = client.TransactionCreate(
		TransactionTypeCancel,
		TransactionObjectTypeDevice,
		deviceId,
		true,
	)
	require.Nil(t, err, "TransactionCreate 'cancel' failed")
	require.NotEqual(t, "", createdTransaction.TransactionId, "Expected a transaction ID")

	waitForPendingTransactions(t, client, deviceId)
}
