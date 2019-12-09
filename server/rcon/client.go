package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	packetIDBadAuth       = -1
	payloadMaxSize        = 1460
	serverdataAuth        = 3
	serverdataExeccommand = 2
)

type payload struct {
	packetID   int32  // 4 bytes
	packetType int32  // 4 bytes
	packetBody []byte // Varies
}

// Client is an RCON client based around the Valve RCON Protocol, see more about the protocol in the
// Valve Wiki: https://developer.valvesoftware.com/wiki/Source_RCON_Protocol
type Client struct {
	connection net.Conn
	password   string
}

var log = logrus.New()

// Both requests and responses are sent as TCP packets. Their payload follows the following basic structure:
// Field        	Type                               value
// Size	         32-bit little-endian Signed Integer
// ID	         32-bit little-endian Signed Integer
// Type          32-bit little-endian Signed Integer
// Body	         Null-terminated ASCII String
// 2-byte pad   Null-terminated ASCII String	        0x00
func (p *payload) calculatePacketSize() int32 {
	return int32(len(p.packetBody) + 4 + 4 + 2)
}

func connectRCON() (*Client, error) {
	address := fmt.Sprintf("%s:%d", viper.GetString("RCONServer"), viper.GetInt("RCONPort"))

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, err
	}

	client := new(Client)
	client.connection = conn
	client.password = viper.GetString("RCONPassword")

	err = client.sendAuthentication(client.password)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewClient contsurct a RCON client againest a running game server and
// issue a ininial authentication using password
func NewClient(host string, port int, pass string) (*Client, error) {
	client, err := connectRCON()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) sendAuthentication(pass string) error {
	payload := createPayload(serverdataAuth, pass)

	_, err := c.sendPayload(payload)
	if err != nil {
		return err
	}

	return nil
}

// SendCommand issues command against running game server
func (c *Client) SendCommand(command string) (string, error) {
	pl := createPayload(serverdataExeccommand, command)
	var response *payload
	response, err := c.sendPayload(pl)
	if err != nil {
		// try to reconnect to remote game server when connection drops
		reconnected := false
		for i := 1; i <= 3; i++ {
			log.Infof("Reconnect to RCON [%d/3]", i)
			newClient, e := connectRCON()
			if e != nil {
				time.Sleep(5 * time.Second)
				continue
			} else {
				c.connection = newClient.connection
				reconnected = true
				response, e = c.sendPayload(pl)
				if e != nil {
					return "", e
				}
				break
			}
		}
		if !reconnected {
			return "", errors.New("Unable to reconnect to RCON server. Game server is down")
		}
	}

	// Trim null bytes
	response.packetBody = bytes.Trim(response.packetBody, "\x00")

	return strings.TrimSpace(string(response.packetBody)), nil
}

func (c *Client) sendPayload(request *payload) (*payload, error) {
	packet, err := createPacketFromPayload(request)
	if err != nil {
		return nil, err
	}

	_, err = c.connection.Write(packet)
	if err != nil {
		return nil, err
	}

	response, err := createPayloadFromPacket(c.connection)
	if err != nil {
		return nil, err
	}

	if response.packetID == packetIDBadAuth {
		return nil, errors.New("Authentication unsuccessful")
	}

	return response, nil
}

// Write packet to the connection as payload struct
func createPacketFromPayload(payload *payload) ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, v := range []interface{}{
		payload.calculatePacketSize(), //Length
		payload.packetID,              //Request ID
		payload.packetType,            //Type
		payload.packetBody,            //Payload
		[]byte{0, 0},                  //pad
	} {
		err := binary.Write(buf, binary.LittleEndian, v)
		if err != nil {
			return nil, errors.New("Unable to write create packet from payload")
		}
	}
	if buf.Len() >= payloadMaxSize {
		return nil, fmt.Errorf("payload exceeded maximum allowed size of %d", payloadMaxSize)
	}

	return buf.Bytes(), nil
}

func createPayload(packetType int, body string) *payload {
	return &payload{
		packetID:   rand.Int31(),
		packetType: int32(packetType),
		packetBody: []byte(body),
	}
}

// Read packet coming from the connection and construct the response object (also as a payload struct)
func createPayloadFromPacket(packetReader io.Reader) (*payload, error) {
	//read packet length
	var packetLength int32
	err := binary.Read(packetReader, binary.LittleEndian, &packetLength)
	if err != nil {
		return nil, errors.New("Unable to read packet length")
	}
	buf := make([]byte, packetLength)
	err = binary.Read(packetReader, binary.LittleEndian, &buf)
	if err != nil {
		err = fmt.Errorf("read packet body fail: %v", err)
		return nil, err
	}
	// check length
	if packetLength < 4+4+2 {
		err = errors.New("packet too short")
		return nil, err
	}
	result := new(payload)
	result.packetID = int32(binary.LittleEndian.Uint32(buf[:4]))
	result.packetType = int32(binary.LittleEndian.Uint32(buf[4:8]))
	result.packetBody = buf[8 : packetLength-2]

	return result, nil
}
