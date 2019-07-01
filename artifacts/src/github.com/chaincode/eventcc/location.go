package eventcc

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/sirupsen/logrus"
)

// Location describes the location data saved by the users
type Location struct {
	ObjectType     string      `json:"docType"`
	User           string      `json:"user"`
	Coordinates    Coordinates `json:"coordinates"`
	Timestamp      string      `json:"timestamp"`
	AdditionalData string      `json:"additional_data"`
	Signature      string      `json:"signature"`
}

// NewLocation creates the default Location object
func NewLocation(user string, lat float64, long float64, timestamp string) Location {
	return Location{"location", user, Coordinates{lat, long},
		timestamp, "", "checked"}
}

// Tracks the given location of the user
func trackLocation(stub shim.ChaincodeStubInterface, args []string, clientID string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("incorrect arguments: expecting a location of form: {'latitute':52,'langitute':10}")
	}

	coordinates := Coordinates{}
	err := json.Unmarshal([]byte(args[0]), &coordinates)
	if err != nil {
		return "", err
	}

	time, err := stub.GetTxTimestamp()
	if err != nil {
		return "", err
	}

	location := Location{ObjectType: "location", Coordinates: coordinates, User: clientID,
		Timestamp: getTimeStamp(time.Seconds)}

	locationAsBytes, _ := json.Marshal(location)

	err = stub.PutState("location-"+getKeyPrefixForUserID(clientID)+"-"+location.Timestamp, locationAsBytes)
	if err != nil {
		return "", fmt.Errorf("Failed to trackLocation: %s", args)
	}
	logrus.Info("Add location: ", string(locationAsBytes))
	return string(locationAsBytes), nil
}

func getLocation(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	if len(args) != 1 {
		return "", errors.New("Incorrect arguments. Expecting a userId")
	}

	value, err := stub.GetState("location-" + getKeyPrefixForUserID(args[0]))
	if err != nil {
		return "", fmt.Errorf("Failed to getLocation for %s with error: %s", args[0], err)
	}
	if value == nil {
		return "", fmt.Errorf("location not found: %s", args[0])
	}
	logrus.Info("Get Location: ", string(value))
	return string(value), nil
}

func getLocations(stub shim.ChaincodeStubInterface, args []string, clientID string) (string, error) {
	if len(args) != 1 {
		return "", errors.New("Incorrect arguments. Expecting a endTime(inclusive)")
	}

	locations, err := getLocationsFromLedgerBefore(stub, args[0], clientID)
	if err != nil {
		return "", err
	}

	locationsAsBytes, err := json.Marshal(locations)
	if err != nil {
		return "", err
	}
	logrus.Info("Get Locations: ", string(locationsAsBytes))
	return string(locationsAsBytes), nil
}

func getLocationsFromLedgerBefore(stub shim.ChaincodeStubInterface, endTime string, clientID string) ([]Location, error) {
	t, err := getEarliestConsideredTime(endTime)
	if err != nil {
		return nil, err
	}

	tLate, err := getLatestConsideredTime(endTime)
	if err != nil {
		return nil, err
	}

	logrus.Infof("query ledger with startKey: %s and endKey: %s", "location-"+getKeyPrefixForUserID(clientID)+"-"+t,
		"location-"+getKeyPrefixForUserID(clientID)+"-"+tLate)
	iter, err := stub.GetStateByRange("location-"+getKeyPrefixForUserID(clientID)+"-"+t,
		"location-"+getKeyPrefixForUserID(clientID)+"-"+tLate)
	defer iter.Close()
	if err != nil {
		return nil, err
	}
	var ledgerLocations []Location
	for iter.HasNext() {
		l, err := iter.Next()
		if err != nil {
			return nil, err
		}
		currentLocation := Location{}
		err = json.Unmarshal(l.Value, &currentLocation)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal location")
		}
		ledgerLocations = append(ledgerLocations, currentLocation)
	}
	return ledgerLocations, nil
}
