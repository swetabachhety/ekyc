/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"errors"
	"fmt"
	"strconv"
	"encoding/json"
	"time"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

var marbleIndexStr = "_marbleindex"				//name for the key/value that will store a list of marbles
var openTradesStr = "_opentrades"				//name for the key/value that will store all open trades

var bankIndexStr = "_allBank"				//name for the key/value that will store a list of all added banks

type Ekyc struct{								//the fieldtags are needed to keep case from bouncing around
	AadharNum string `json:"aadharNum"`					//name is replaced with aadharNum
	Timestamp int64 `json:"timestamp"`			//utc timestamp of creation		//color is replaced with time stamp
	Size int `json:"size"`
	User string `json:"user"`
}

type Description struct{
	Timestamp int64 `json:"timestamp"`					//color is replaced with time stamp
	Size int `json:"size"`
}

type AnOpenTrade struct{
	User string `json:"user"`					//user who created the open trade order
	Timestamp int64 `json:"timestamp"`			//utc timestamp of creation
	Want Description  `json:"want"`				//description of desired marble
	Willing []Description `json:"willing"`		//array of marbles willing to trade away
}

type AllTrades struct{
	OpenTrades []AnOpenTrade `json:"open_trades"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode ::: %s", err)
	}
}

// ============================================================================================================================
// Init - reset all the things
// ============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments....Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("=========Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("kyc", []byte(strconv.Itoa(Aval)))				//making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}
	
	// Write the state to the ledger
	var invokeArgs  string 
	invokeArgs = "a"
	err = stub.PutState("bank", []byte(invokeArgs))				
	if err != nil {
		return nil, err
	}
	
	var emptyList []string
	bankListAsBytes, _ := json.Marshal(emptyList)								//marshal an emtpy array of strings to clear the index
	err = stub.PutState(bankIndexStr, bankListAsBytes)
	if err != nil {
		return nil, err
	}
	
	var empty []string
	jsonAsBytes, _ := json.Marshal(empty)								//marshal an emtpy array of strings to clear the index
	err = stub.PutState(marbleIndexStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}
	
	var trades AllTrades
	jsonAsBytes, _ = json.Marshal(trades)								//clear the open trade struct
	err = stub.PutState(openTradesStr, jsonAsBytes)
	if err != nil {
		return nil, err
	}
	
	return nil, nil
}

// ============================================================================================================================
// Run - Our entry point for Invocations - [LEGACY] obc-peer 4/25/2016
// ============================================================================================================================
func (t *SimpleChaincode) Run(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("=======run is running " + function)
	return t.Invoke(stub, function, args)
}

// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("==========invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "delete" {										//deletes an entity from its state
		res, err := t.Delete(stub, args)
		cleanTrades(stub)													//lets make sure all open trades are still valid
		return res, err
	} else if function == "write" {											//writes a value to the chaincode state
		return t.Write(stub, args)
	} else if function == "writeBank" {											//writes a value to the chaincode state
		return t.WriteBank(stub, args)
	} else if function == "init_marble" {									//create a new marble
		return t.init_marble(stub, args)
	} else if function == "set_user" {										//change owner of a marble
		res, err := t.set_user(stub, args)
		cleanTrades(stub)													//lets make sure all open trades are still valid
		return res, err
	} else if function == "open_trade" {									//create a new trade order
		return t.open_trade(stub, args)
	} else if function == "perform_trade" {									//forfill an open trade order
		res, err := t.perform_trade(stub, args)
		cleanTrades(stub)													//lets clean just in case
		return res, err
	} else if function == "remove_trade" {									//cancel an open trade order
		return t.remove_trade(stub, args)
	}
	fmt.Println("========invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation")
}

// ============================================================================================================================
// Query - Our entry point for Queries
// ============================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("=======query is running " + function)

	// Handle different functions
	if function == "read" {		
		return t.read(stub, args)	//read a KYC
	}else if function == "readBank" {		
		return t.readBank(stub, args)	//read a Bank details
	}else if function == "readAll" {		
		return t.readAll(stub, args)	//read list of registered Banks
	}
	fmt.Println("=======query did not find func: " + function)						//error

	return nil, errors.New("Received unknown function query=Query")
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var aadharNum, jsonResp string
	//var fail Ekyc;

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting AadharNum to be queried")
	}

	aadharNum = args[0]
	valAsbytes, err := stub.GetState(aadharNum)									//get the var from chaincode state
	
	//infoAsBytes, err := stub.GetState(aadharNum)
	//if err != nil {
	//	return nil, errors.New("Failed to get aadharNum")
	//}
	//res := Ekyc{}
	//json.Unmarshal(infoAsBytes, &res)
	
	//var  valAsString string 
	//valAsString = "Financial Institue : " + res.User + " timestamp : " + strconv.FormatInt(res.Timestamp,16) + " Size : " + //strconv.Itoa(res.Size)
	//valAsbytes=res.User+".."+res.Timestamp
	
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + aadharNum + "\"}"
		return nil, errors.New(jsonResp)
	}
    return valAsbytes, nil	
	
	//return []byte(valAsString), nil													//send it onward
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) readBank(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var bankName, jsonResp string
	//var fail Ekyc;

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting bankName to be queried")
	}

	bankName = args[0]
	valAsbytes, err := stub.GetState(bankName)									//get the var from chaincode state
	
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + bankName + "\"}"
		return nil, errors.New(jsonResp)
	}
    return valAsbytes, nil	
	
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) readAll(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var jsonResp string

	valAsbytes, err := stub.GetState(bankIndexStr)									//get the var from chaincode state
	
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for list of registered banks \"}"
		return nil, errors.New(jsonResp)
	}
    return valAsbytes, nil	
	
}

// ============================================================================================================================
// Delete - remove a key/value pair from state
// ============================================================================================================================
func (t *SimpleChaincode) Delete(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
	
	aadharNum := args[0]
	err := stub.DelState(aadharNum)													//remove the key from chaincode state
	if err != nil {
		return nil, errors.New("Failed to delete aadharNum")
	}

	//get the marble index
	marblesAsBytes, err := stub.GetState(marbleIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get KYC index")
	}
	var marbleIndex []string
	json.Unmarshal(marblesAsBytes, &marbleIndex)								//un stringify it aka JSON.parse()
	
	//remove marble from index
	for i,val := range marbleIndex{
		fmt.Println(strconv.Itoa(i) + " -====== looking at " + val + " for " + aadharNum)
		if val == aadharNum{															//find the correct marble
			fmt.Println("found KYC")
			marbleIndex = append(marbleIndex[:i], marbleIndex[i+1:]...)			//remove it
			for x:= range marbleIndex{											//debug prints...
				fmt.Println(string(x) + " - " + marbleIndex[x])
			}
			break
		}
	}
	jsonAsBytes, _ := json.Marshal(marbleIndex)									//save new index
	err = stub.PutState(marbleIndexStr, jsonAsBytes)
	return nil, nil
}

// ============================================================================================================================
// Write - write variable into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) Write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var aadharNum, value string // Entities
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("=============Incorrect number of arguments. Expecting 2. aadharNum of the variable and value to set")
	}
	
	//addEykc := Ekyc{}
	//addEykc.AadharNum = args[0]
	//addEykc.Timestamp = makeTimestamp()											//use timestamp as an ID
	//addEykc.User = args[1]
	//size, err := strconv.Atoi(args[2])
	//addEykc.Size= size
	//fmt.Println("- Perform Ekyc for aadhar Number")
	//jsonAsBytes, _ := json.Marshal(addEykc)
	//err = stub.PutState("aadharNum", jsonAsBytes)
	
	aadharNum = args[0]															
	value = args[1] + ";" + time.Now().Format(time.RFC850)
	err = stub.PutState(aadharNum, []byte(value))								//write the variable into the chaincode state
	
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// Write - write Bank realated data into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) WriteBank(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var bankName, value string // Entities
	var err error
	var err1 error
	//var valAsbytes []byte
	var valueAll string
	fmt.Println("running writeBank()")

	if len(args) != 4 {
		return nil, errors.New("=============Incorrect number of arguments. Expecting 2. aadharNum of the variable and value to set")
	}	
	
	bankName = args[0]															
	value = args[1] + ";" + args[2] + ";" + args[3] 
	err = stub.PutState(bankName, []byte(value))								//write the variable into the chaincode state
	
	if err != nil {
		return nil, err
	}
	
	valAsbytes, errr := t.readAll(stub, args)
	if errr != nil {
		return nil, errr
	}
																
	valueAll = string(valAsbytes) + ";" + args[0]
	err1 = stub.PutState(bankIndexStr, []byte(valueAll))	
	if err1 != nil {
		return nil, err
	}
	
	return nil, nil
}

// ============================================================================================================================
// Init Marble - create a new marble, store into chaincode state
// ============================================================================================================================
func (t *SimpleChaincode) init_marble(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	//   0       1       2     3
	// "asdf", "blue", "35", "bob"
	if len(args) != 4 {
		return nil, errors.New("========Incorrect number of arguments. Expecting 4")
	}

	//input sanitation
	fmt.Println("- start init marble")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return nil, errors.New("4th argument must be a non-empty string")
	}
	aadharNum := args[0]
	timestamp := strings.ToLower(args[1])
	user := strings.ToLower(args[3])
	size, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("3rd argument must be a numeric string")
	}

	//check if aadhar Number already exists
	marbleAsBytes, err := stub.GetState(aadharNum)
	if err != nil {
		return nil, errors.New("Failed to get aadharNum")
	}
	res := Ekyc{}
	json.Unmarshal(marbleAsBytes, &res)
	if res.AadharNum == aadharNum{
		fmt.Println("eKYC for this Aadhar number is arleady done: " + aadharNum)
		fmt.Println(res);
		return nil, errors.New("Aadhar number arleady exists")				//all stop if aadharNum already exists
	}
	
	//build the marble json string manually
	str := `{"aadharNum": "` + aadharNum + `", "timestamp": "` + timestamp + `", "size": ` + strconv.Itoa(size) + `, "user": "` + user + `"}`
	err = stub.PutState(aadharNum, []byte(str))									//store marble with id as key
	if err != nil {
		return nil, err
	}
		
	//get the marble index
	marblesAsBytes, err := stub.GetState(marbleIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var marbleIndex []string
	json.Unmarshal(marblesAsBytes, &marbleIndex)							//un stringify it aka JSON.parse()
	
	//append
	marbleIndex = append(marbleIndex, aadharNum)									//add Aaadhar Number to index list
	fmt.Println("! marble index: ", marbleIndex)
	jsonAsBytes, _ := json.Marshal(marbleIndex)
	err = stub.PutState(marbleIndexStr, jsonAsBytes)						//store Aadhar Number

	fmt.Println("-====== end init marble")
	return nil, nil
}

// ============================================================================================================================
// Set User Permission on Marble
// ============================================================================================================================
func (t *SimpleChaincode) set_user(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	
	//   0       1
	// "aadharNum", "bob"
	if len(args) < 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2")
	}
	
	fmt.Println("- start set user")
	fmt.Println(args[0] + " - " + args[1])
	marbleAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return nil, errors.New("Failed to get thing")
	}
	res := Ekyc{}
	json.Unmarshal(marbleAsBytes, &res)										//un stringify it aka JSON.parse()
	res.User = args[1]														//change the user
	
	jsonAsBytes, _ := json.Marshal(res)
	err = stub.PutState(args[0], jsonAsBytes)								//rewrite the marble with id as key
	if err != nil {
		return nil, err
	}
	
	fmt.Println("- end set user")
	return nil, nil
}

// ============================================================================================================================
// Open Trade - create an open trade for a marble you want with marbles you have 
// ============================================================================================================================
func (t *SimpleChaincode) open_trade(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	var will_size int
	var trade_away Description
	
	//	0        1      2     3      4      5       6
	//["bob", "blue", "16", "red", "16"] *"blue", "35*
	if len(args) < 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting like 5?")
	}
	if len(args)%2 == 0{
		return nil, errors.New("Incorrect number of arguments. Expecting an odd number")
	}

	size1, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("3rd argument must be a numeric string")
	}

	open := AnOpenTrade{}
	open.User = args[0]
	open.Timestamp = makeTimestamp()											//use timestamp as an ID
	timestamp, err := strconv.ParseInt(args[0], 10, 64)
	open.Want.Timestamp = timestamp
	open.Want.Size =  size1
	fmt.Println("- start open trade")
	jsonAsBytes, _ := json.Marshal(open)
	err = stub.PutState("_debug1", jsonAsBytes)

	for i:=3; i < len(args); i++ {												//create and append each willing trade
		will_size, err = strconv.Atoi(args[i + 1])
		if err != nil {
			msg := "is not a numeric string " + args[i + 1]
			fmt.Println(msg)
			return nil, errors.New(msg)
		}
		
		trade_away = Description{}
		timestamp1, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Println(err)
		}
		trade_away.Timestamp = timestamp1
		trade_away.Size =  will_size
		fmt.Println("! created trade_away: " + args[i])
		jsonAsBytes, _ = json.Marshal(trade_away)
		err = stub.PutState("_debug2", jsonAsBytes)
		
		open.Willing = append(open.Willing, trade_away)
		fmt.Println("! appended willing to open")
		i++;
	}
	
	//get the open trade struct
	tradesAsBytes, err := stub.GetState(openTradesStr)
	if err != nil {
		return nil, errors.New("Failed to get opentrades")
	}
	var trades AllTrades
	json.Unmarshal(tradesAsBytes, &trades)										//un stringify it aka JSON.parse()
	
	trades.OpenTrades = append(trades.OpenTrades, open);						//append to open trades
	fmt.Println("! appended open to trades")
	jsonAsBytes, _ = json.Marshal(trades)
	err = stub.PutState(openTradesStr, jsonAsBytes)								//rewrite open orders
	if err != nil {
		return nil, err
	}
	fmt.Println("- end open trade")
	return nil, nil
}

// ============================================================================================================================
// Perform Trade - close an open trade and move ownership
// ============================================================================================================================
func (t *SimpleChaincode) perform_trade(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	
	//	0		1					2					3				4					5
	//[data.id, data.closer.user, data.closer.aadharNum, data.opener.user, data.opener.timestamp, data.opener.size]
	if len(args) < 6 {
		return nil, errors.New("Incorrect number of arguments. Expecting 6")
	}
	
	fmt.Println("- start close trade")
	timestamp, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, errors.New("1st argument must be a numeric string")
	}
	
	size, err := strconv.Atoi(args[5])
	if err != nil {
		return nil, errors.New("6th argument must be a numeric string")
	}
	
	//get the open trade struct
	tradesAsBytes, err := stub.GetState(openTradesStr)
	if err != nil {
		return nil, errors.New("Failed to get opentrades")
	}
	var trades AllTrades
	json.Unmarshal(tradesAsBytes, &trades)															//un stringify it aka JSON.parse()
	
	for i := range trades.OpenTrades{																//look for the trade
		fmt.Println("looking at " + strconv.FormatInt(trades.OpenTrades[i].Timestamp, 10) + " for " + strconv.FormatInt(timestamp, 10))
		if trades.OpenTrades[i].Timestamp == timestamp{
			fmt.Println("found the trade");
			
			
			marbleAsBytes, err := stub.GetState(args[2])
			if err != nil {
				return nil, errors.New("Failed to get thing")
			}
			closersMarble := Ekyc{}
			json.Unmarshal(marbleAsBytes, &closersMarble)											//un stringify it aka JSON.parse()
			
			//verify if marble meets trade requirements
			//if closersMarble.Timestamp != trades.OpenTrades[i].Want.Timestamp || closersMarble.Size != trades.OpenTrades[i].Want.Size {
			if  closersMarble.Size != trades.OpenTrades[i].Want.Size {			
				msg := "marble in input does not meet trade requriements"
				fmt.Println(msg)
				return nil, errors.New(msg)
			}
			
			timestamp, err := strconv.ParseInt(args[4], 10, 64)
			ekyc, e := findMarble4Trade(stub, trades.OpenTrades[i].User, timestamp, size)			//find a marble that is suitable from opener
			if(e == nil){
				fmt.Println("! no errors, proceeding")

				t.set_user(stub, []string{args[2], trades.OpenTrades[i].User})						//change owner of selected marble, closer -> opener
				t.set_user(stub, []string{ekyc.AadharNum, args[1]})									//change owner of selected marble, opener -> closer
			
				trades.OpenTrades = append(trades.OpenTrades[:i], trades.OpenTrades[i+1:]...)		//remove trade
				jsonAsBytes, _ := json.Marshal(trades)
				err = stub.PutState(openTradesStr, jsonAsBytes)										//rewrite open orders
				if err != nil {
					return nil, err
				}
			}
		}
	}
	fmt.Println("- end close trade")
	return nil, nil
}

// ============================================================================================================================
// findMarble4Trade - look for a matching marble that this user owns and return it
// ============================================================================================================================
func findMarble4Trade(stub shim.ChaincodeStubInterface, user string, timestamp int64, size int )(m Ekyc, err error){
	var fail Ekyc;
	fmt.Println("- start find marble 4 trade")
	//fmt.Println("looking for " + user + ", " + timestamp + ", " + strconv.Itoa(size));

	//get the marble index
	marblesAsBytes, err := stub.GetState(marbleIndexStr)
	if err != nil {
		return fail, errors.New("Failed to get marble index")
	}
	var marbleIndex []string
	json.Unmarshal(marblesAsBytes, &marbleIndex)								//un stringify it aka JSON.parse()
	
	for i:= range marbleIndex{													//iter through all the marbles
		//fmt.Println("looking @ AadharNum: " + marbleIndex[i]);

		marbleAsBytes, err := stub.GetState(marbleIndex[i])						//grab this marble
		if err != nil {
			return fail, errors.New("Failed to get marble")
		}
		res := Ekyc{}
		json.Unmarshal(marbleAsBytes, &res)										//un stringify it aka JSON.parse()
		//fmt.Println("looking @ " + res.User + ", " + res.Timestamp + ", " + strconv.Itoa(res.Size));
		
		//check for user && timestamp && size
		if strings.ToLower(res.User) == strings.ToLower(user) && res.Timestamp == timestamp && res.Size == size{
			fmt.Println("found a marble: " + res.AadharNum)
			fmt.Println("! end find marble 4 trade")
			return res, nil
		}
	}
	
	fmt.Println("- end find marble 4 trade - error")
	return fail, errors.New("Did not find marble to use in this trade")
}

// ============================================================================================================================
// Make Timestamp - create a timestamp in ms
// ============================================================================================================================
func makeTimestamp() int64 {
    return time.Now().UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond))
}

// ============================================================================================================================
// Remove Open Trade - close an open trade
// ============================================================================================================================
func (t *SimpleChaincode) remove_trade(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	
	//	0
	//[data.id]
	if len(args) < 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
	
	fmt.Println("- start remove trade")
	timestamp, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, errors.New("1st argument must be a numeric string")
	}
	
	//get the open trade struct
	tradesAsBytes, err := stub.GetState(openTradesStr)
	if err != nil {
		return nil, errors.New("Failed to get opentrades")
	}
	var trades AllTrades
	json.Unmarshal(tradesAsBytes, &trades)																//un stringify it aka JSON.parse()
	
	for i := range trades.OpenTrades{																	//look for the trade
		//fmt.Println("looking at " + strconv.FormatInt(trades.OpenTrades[i].Timestamp, 10) + " for " + strconv.FormatInt(timestamp, 10))
		if trades.OpenTrades[i].Timestamp == timestamp{
			fmt.Println("found the trade");
			trades.OpenTrades = append(trades.OpenTrades[:i], trades.OpenTrades[i+1:]...)				//remove this trade
			jsonAsBytes, _ := json.Marshal(trades)
			err = stub.PutState(openTradesStr, jsonAsBytes)												//rewrite open orders
			if err != nil {
				return nil, err
			}
			break
		}
	}
	
	fmt.Println("- end remove trade")
	return nil, nil
}

// ============================================================================================================================
// Clean Up Open Trades - make sure open trades are still possible, remove choices that are no longer possible, remove trades that have no valid choices
// ============================================================================================================================
func cleanTrades(stub shim.ChaincodeStubInterface)(err error){
	var didWork = false
	fmt.Println("- start clean trades")
	
	//get the open trade struct
	tradesAsBytes, err := stub.GetState(openTradesStr)
	if err != nil {
		return errors.New("Failed to get opentrades")
	}
	var trades AllTrades
	json.Unmarshal(tradesAsBytes, &trades)																		//un stringify it aka JSON.parse()
	
	fmt.Println("# trades " + strconv.Itoa(len(trades.OpenTrades)))
	for i:=0; i<len(trades.OpenTrades); {																		//iter over all the known open trades
		fmt.Println(strconv.Itoa(i) + ": looking at trade " + strconv.FormatInt(trades.OpenTrades[i].Timestamp, 10))
		
		fmt.Println("# options " + strconv.Itoa(len(trades.OpenTrades[i].Willing)))
		for x:=0; x<len(trades.OpenTrades[i].Willing); {														//find a marble that is suitable
			fmt.Println("! on next option " + strconv.Itoa(i) + ":" + strconv.Itoa(x))
			_, e := findMarble4Trade(stub, trades.OpenTrades[i].User, trades.OpenTrades[i].Willing[x].Timestamp, trades.OpenTrades[i].Willing[x].Size)
			if(e != nil){
				fmt.Println("! errors with this option, removing option")
				didWork = true
				trades.OpenTrades[i].Willing = append(trades.OpenTrades[i].Willing[:x], trades.OpenTrades[i].Willing[x+1:]...)	//remove this option
				x--;
			}else{
				fmt.Println("! this option is fine")
			}
			
			x++
			fmt.Println("! x:" + strconv.Itoa(x))
			if x >= len(trades.OpenTrades[i].Willing) {														//things might have shifted, recalcuate
				break
			}
		}
		
		if len(trades.OpenTrades[i].Willing) == 0 {
			fmt.Println("! no more options for this trade, removing trade")
			didWork = true
			trades.OpenTrades = append(trades.OpenTrades[:i], trades.OpenTrades[i+1:]...)					//remove this trade
			i--;
		}
		
		i++
		fmt.Println("! i:" + strconv.Itoa(i))
		if i >= len(trades.OpenTrades) {																	//things might have shifted, recalcuate
			break
		}
	}

	if(didWork){
		fmt.Println("! saving open trade changes")
		jsonAsBytes, _ := json.Marshal(trades)
		err = stub.PutState(openTradesStr, jsonAsBytes)														//rewrite open orders
		if err != nil {
			return err
		}
	}else{
		fmt.Println("! all open trades are fine")
	}

	fmt.Println("- end clean trades")
	return nil
}