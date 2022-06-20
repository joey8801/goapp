package main

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

const (
	channelId     = "mychannel"
	chaincodeName = "basic"
	address       = "127.0.0.1:5050"
)

type Asset struct {
	ID             string `json:"id"`
	Color          string `json:"color"`
	Size           string `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue string `json:"appraisedValue"`
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/api/createasset", createAsset)

	return r
}

func createAsset(c *gin.Context) {
	notifier := make(chan string)
	contract := setup()
	result := ""

	go func(notifier chan string) {
		asset := &Asset{}
		value, _ := ioutil.ReadAll(c.Request.Body)

		str3 := bytes.NewBuffer([]byte(value)).String()
		json.Unmarshal([]byte(str3), &asset)

		resp, err := contract.SubmitTransaction("CreateAsset", asset.ID, asset.Color, asset.Size, asset.Owner, asset.AppraisedValue)
		log.Println(resp)

		if err != nil {
			notifier <- "Failed to execute" + err.Error()
		} else {
			notifier <- string(resp)
		}
		log.Println("15")

	}(notifier)
	result = <-notifier

	c.JSON(200, result)
}

func main() {
	r := setupRouter()
	r.Run(":8001")
}

func populateWallet(wallet *gateway.Wallet) error {
	log.Println("============ Populating wallet ============")
	credPath := filepath.Join(
		"..",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"users",
		"User1@org1.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := ioutil.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org1MSP", string(cert), string(key))

	return wallet.Put("appUser", identity)
}

func setup() *gateway.Contract {
	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			log.Fatalf("Failed to populate wallet contents: %v", err)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"connection-org1.yaml",
	)

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gw.Close()

	network, err := gw.GetNetwork("mychannel")
	if err != nil {
		log.Fatalf("Failed to get network: %v", err)
	}

	contract := network.GetContract("basic")

	return contract
}
