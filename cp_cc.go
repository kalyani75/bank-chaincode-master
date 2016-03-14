package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/openblockchain/obc-peer/openchain/chaincode/shim"
)

var cpPrefix = "cp:"
var accountPrefix = "acct:"
var accountsKey = "accounts"

var recentLeapYear = 2016

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func generateCUSIPSuffix(issueDate string, days int) (string, error) {

	t, err := msToTime(issueDate)
	if err != nil {
		return "", err
	}

	maturityDate := t.AddDate(0, 0, days)
	month := int(maturityDate.Month())
	day := maturityDate.Day()

	suffix := seventhDigit[month] + eigthDigit[day]
	return suffix, nil

}

const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond), nil
}



type Owner struct {
	Company string    `json:"company"`
}

type CP struct {
	CUSIP     string  `json:"cusip"`
	Ticker    string  `json:"ticker"`
	Par       float64 `json:"par"`
	Owners    []Owner `json:"owner"`
	Issuer    string  `json:"issuer"`
	IssueDate string  `json:"issueDate"`
}

type Account struct {
	ID          string  `json:"id"`
	Prefix      string  `json:"prefix"`
	CashBalance float64 `json:"cashBalance"`
	AssetsIds   []string `json:"assetIds"`
}

type Transaction struct {
	CUSIP       string   `json:"cusip"`
	FromCompany string   `json:"fromCompany"`
	ToCompany   string   `json:"toCompany"`
}

func (t *SimpleChaincode) createAccounts(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//  				0
	// "number of accounts to create"
	var err error
	numAccounts, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("error creating accounts with input")
		return nil, errors.New("createAccounts accepts a single integer argument")
	}
	//create a bunch of accounts
	var account Account
	counter := 1
	for counter <= numAccounts {
		var prefix string
		suffix := "000A"
		if counter < 10 {
			prefix = strconv.Itoa(counter) + "0" + suffix
		} else {
			prefix = strconv.Itoa(counter) + suffix
		}
		var assetIds []string
		account = Account{ID: "company" + strconv.Itoa(counter), Prefix: prefix, CashBalance: 10000000.0, AssetsIds: assetIds}
		accountBytes, err := json.Marshal(&account)
		if err != nil {
			fmt.Println("error creating account" + account.ID)
			return nil, errors.New("Error creating account " + account.ID)
		}
		err = stub.PutState(accountPrefix+account.ID, accountBytes)
		counter++
		fmt.Println("created account" + accountPrefix + account.ID)
	}
	var blank []string
	blankBytes, _ := json.Marshal(&blank)
	err = stub.PutState("PaperKeys", blankBytes)

	fmt.Println("Accounts created")
	return nil, nil

}
func (t *SimpleChaincode) issueCommercialPaper(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	/*		0
		json
	  	{
			"ticker":  "string",
			"par": 0.00,
			"qty": 10,
			"discount": 7.5,
			"maturity": 30,
			"owners": [ // This one is not required
				{
					"company": "company1",
					"quantity": 5
				},
				{
					"company": "company3",
					"quantity": 3
				},
				{
					"company": "company4",
					"quantity": 2
				}
			],				
			"issuer":"company2",
			"issueDate":"1456161763790"  (current time in milliseconds as a string)

		}
	*/
	//need one arg
	if len(args) != 1 {
		fmt.Println("error invalid arguments")
		return nil, errors.New("Incorrect number of arguments. Expecting cheque record")
	}

	var cq Cheque
	var err error
	var account Account

	fmt.Println("Unmarshalling Cheque");
	err = json.Unmarshal([]byte(args[0]), &cq)
	if err != nil {
		fmt.Println("error invalid paper issue")
		return nil, errors.New("Invalid commercial paper issue")
	}

	//generate the CUSIP
	//get account prefix
	fmt.Println("Getting state of - " + accountPrefix + cq.Issuer);
	accountBytes, err := stub.GetState(accountPrefix + cq.Issuer);
	if err != nil {
		fmt.Println("Error Getting state of - " + accountPrefix + cq.Issuer);
		return nil, errors.New("Error retrieving account " + cq.Issuer)
	}
	err = json.Unmarshal(accountBytes, &account)
	if err != nil {
		fmt.Println("Error Unmarshalling accountBytes");
		return nil, errors.New("Error retrieving account " + cq.Issuer)
	}
	fmt.Println("KD After Get State" + account)
	account.AssetsIds = append(account.AssetsIds, cq.CUSIP)

	// Set the issuer to be the owner of all quantity
	var owner Owner;
	owner.Company = cq.Issuer
	//owner.Quantity = cq.Qty
	
	cq.Owners = append(cq.Owners, owner)


	//commented by KD
	/*suffix, err := generateCUSIPSuffix(cq.IssueDate, cq.Maturity)
	if err != nil {
		fmt.Println("Error generating cusip");
		return nil, errors.New("Error generating CUSIP")
	}*/

	fmt.Println("Marshalling cq bytes");
	//cq.CUSIP = account.Prefix + suffix
	rand.Seed(time.Now().UTC().UnixNano())
	cq.CUSIP = randomString(10)

	fmt.Println("Getting State on cq " + cq.CUSIP)
	cqRxBytes, err := stub.GetState(cqPrefix+cq.CUSIP);
	if cqRxBytes == nil {
		fmt.Println("CUSIP does not exist, creating it")
		cqBytes, err := json.Marshal(&cq)
		if err != nil {
			fmt.Println("Error marshalling cq");
			return nil, errors.New("Error issuing Cheque")
		}
		err = stub.PutState(cqPrefix+cq.CUSIP, cqBytes)
		if err != nil {
			fmt.Println("Error issuing paper");
			return nil, errors.New("Error issuing Cheque")
		}

		fmt.Println("Marshalling account bytes to write");
		accountBytesToWrite, err := json.Marshal(&account)
		if err != nil {
			fmt.Println("Error marshalling account");
			return nil, errors.New("Error issuing Cheque")
		}
		err = stub.PutState(accountPrefix + cq.Issuer, accountBytesToWrite)
		if err != nil {
			fmt.Println("Error putting state on accountBytesToWrite");
			return nil, errors.New("Error issuing Cheque")
		}
		
		
		// Update the paper keys by adding the new key
		fmt.Println("Getting Cheque Keys");
		keysBytes, err := stub.GetState("PaperKeys")
		if err != nil {
			fmt.Println("Error retrieving paper keys");
			return nil, errors.New("Error retrieving Cheque keys")
		}
		var keys []string
		err = json.Unmarshal(keysBytes, &keys)
		if err != nil {
			fmt.Println("Error unmarshel keys");
			return nil, errors.New("Error unmarshalling Cheque keys ")
		}
		
		fmt.Println("Appending the new key to Cheque Keys");
		foundKey := false
		for _, key := range keys {
			if key == cqPrefix+cq.CUSIP {
				foundKey = true
			}
		}
		if foundKey == false {
			keys = append(keys, cqPrefix+cq.CUSIP);		
			keysBytesToWrite, err := json.Marshal(&keys)
			if err != nil {
				fmt.Println("Error marshalling keys");
				return nil, errors.New("Error marshalling the keys")
			}
			fmt.Println("Put state on PaperKeys");
			err = stub.PutState("PaperKeys", keysBytesToWrite)
			if err != nil {
				fmt.Println("Error writting keys back");
				return nil, errors.New("Error writing the keys back")
			}
		}
		
		fmt.Println("Issue commercial paper %+v\n", cq)
		return nil, nil
	} else {
		fmt.Println("CUSIP exists")
		
		var cqrx Cheque
		fmt.Println("Unmarshalling Cheque " + cq.CUSIP)
		err = json.Unmarshal(cpRxBytes, &cqrx)
		if err != nil {
			fmt.Println("Error unmarshalling cq " + cq.CUSIP)
			return nil, errors.New("Error unmarshalling cq " + cq.CUSIP)
		}
		//commented by KD
		/*cqrx.Qty = cqrx.Qty + cq.Qty;
		
		for key, val := range cqrx.Owners {
			if val.Company == cq.Issuer {
				cqrx.Owners[key].Quantity += cq.Qty
				break
			}
		}*/
				
		cpWriteBytes, err := json.Marshal(&cqrx)
		if err != nil {
			fmt.Println("Error marshalling cq");
			return nil, errors.New("Error issuing commercial paper")
		}
		err = stub.PutState(cqPrefix+cq.CUSIP, cpWriteBytes)
		if err != nil {
			fmt.Println("Error issuing paper");
			return nil, errors.New("Error issuing commercial paper")
		}

		fmt.Println("Updated commercial paper %+v\n", cqrx)
		return nil, nil
	}
}
func GetAllCPs(stub *shim.ChaincodeStub) ([]CP, error){
	
	var allCPs []CP
	
	// Get list of all the keys
	keysBytes, err := stub.GetState("PaperKeys")
	if err != nil {
		fmt.Println("Error retrieving paper keys")
		return nil, errors.New("Error retrieving paper keys")
	}
	var keys []string
	err = json.Unmarshal(keysBytes, &keys)
	if err != nil {
		fmt.Println("Error unmarshalling paper keys")
		return nil, errors.New("Error unmarshalling paper keys")
	}

	// Get all the cps
	for _, value := range keys {
		cpBytes, err := stub.GetState(value);
		
		var cp CP
		err = json.Unmarshal(cpBytes, &cp)
		if err != nil {
			fmt.Println("Error retrieving cp " + value)
			return nil, errors.New("Error retrieving cp " + value)
		}
		
		fmt.Println("Appending CP" + value)
		allCPs = append(allCPs, cp)
	}	
	
	return allCPs, nil
}

func GetCP(cpid string, stub *shim.ChaincodeStub) (CP, error){
	var cp CP

	cpBytes, err := stub.GetState(cpid);
	if err != nil {
		fmt.Println("Error retrieving cp " + cpid)
		return cp, errors.New("Error retrieving cp " + cpid)
	}
		
	err = json.Unmarshal(cpBytes, &cp)
	if err != nil {
		fmt.Println("Error unmarshalling cp " + cpid)
		return cp, errors.New("Error unmarshalling cp " + cpid)
	}
		
	return cp, nil
}


func GetCompany(companyID string, stub *shim.ChaincodeStub) (Account, error){
	var company Account
	companyBytes, err := stub.GetState(accountPrefix+companyID);
	if err != nil {
		fmt.Println("Account not found " + companyID)
		return company, errors.New("Account not found " + companyID)
	}

	err = json.Unmarshal(companyBytes, &company)
	if err != nil {
		fmt.Println("Error unmarshalling account " + companyID)
		return company, errors.New("Error unmarshalling account " + companyID)
	}
	
	return company, nil
}
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	//need one arg
	if len(args) < 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting ......")
	}

	if args[0] == "GetAllCPs" {
		fmt.Println("Getting all CPs");
		allCPs, err := GetAllCPs(stub);
		if err != nil {
			fmt.Println("Error from getallcps");
			return nil, err
		} else {
			allCPsBytes, err1 := json.Marshal(&allCPs)
			if err1 != nil {
				fmt.Println("Error marshalling allcps");
				return nil, err1
			}	
			fmt.Println("All success, returning allcps");
			return allCPsBytes, nil		 
		}
	} else if args[0] == "GetCP" {
		fmt.Println("Getting particular cp");
		cp, err := GetCP(args[1], stub);
		if err != nil {
			fmt.Println("Error Getting particular cp");
			return nil, err
		} else {
			cpBytes, err1 := json.Marshal(&cp)
			if err1 != nil {
				fmt.Println("Error marshalling the cp");
				return nil, err1
			}	
			fmt.Println("All success, returning the cp");
			return cpBytes, nil		 
		}
	} else if args[0] == "GetCompany" {
		fmt.Println("Getting the company");
		company, err := GetCompany(args[1], stub);
		if err != nil {
			fmt.Println("Error from getCompany");
			return nil, err
		} else {
			companyBytes, err1 := json.Marshal(&company)
			if err1 != nil {
				fmt.Println("Error marshalling the company");
				return nil, err1
			}	
			fmt.Println("All success, returning the company");
			return companyBytes, nil		 
		}
	} else {
		fmt.Println("Generic Query call");
		bytes, err := stub.GetState(args[0])

		if err != nil {
			fmt.Println("Some error happenend");
			return nil, errors.New("Some Error happened")
		}

		fmt.Println("All success, returning from generic");
		return bytes, nil		
	}
}

func (t *SimpleChaincode) Run(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	
/*	if function == "issueCommercialPaper" {
		fmt.Println("Firing issueCommercialPaper");
		//Create an asset with some value
		return t.issueCommercialPaper(stub, args)
	} else if function == "transferPaper" {
		fmt.Println("Firing cretransferPaperateAccounts");
		return t.transferPaper(stub, args)
	} else if function == "createAccounts" {
		fmt.Println("Firing createAccounts");
		return t.createAccounts(stub, args)
	}*/
	if function == "issueCommercialPaper" {
		fmt.Println("Firing issueCommercialPaper");
		//Create an asset with some value
		return t.issueCommercialPaper(stub, args)
	} elseif function == "createAccounts" {
		fmt.Println("Firing createAccounts");
		return t.createAccounts(stub, args)
	}
	return nil, errors.New("Received unknown function invocation")
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Println("Error starting Simple chaincode: %s", err)
	}
}

//lookup tables for last two digits of CUSIP
var seventhDigit = map[int]string{
	1:  "A",
	2:  "B",
	3:  "C",
	4:  "D",
	5:  "E",
	6:  "F",
	7:  "G",
	8:  "H",
	9:  "J",
	10: "K",
	11: "L",
	12: "M",
	13: "N",
	14: "P",
	15: "Q",
	16: "R",
	17: "S",
	18: "T",
	19: "U",
	20: "V",
	21: "W",
	22: "X",
	23: "Y",
	24: "Z",
}

var eigthDigit = map[int]string{
	1:  "1",
	2:  "2",
	3:  "3",
	4:  "4",
	5:  "5",
	6:  "6",
	7:  "7",
	8:  "8",
	9:  "9",
	10: "A",
	11: "B",
	12: "C",
	13: "D",
	14: "E",
	15: "F",
	16: "G",
	17: "H",
	18: "J",
	19: "K",
	20: "L",
	21: "M",
	22: "N",
	23: "P",
	24: "Q",
	25: "R",
	26: "S",
	27: "T",
	28: "U",
	29: "V",
	30: "W",
	31: "X",
}
