package frontend

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// Define ioctl constants based on Linux DVB API
const (
	_FE_READ_SIGNAL_STRENGTH    = 0x80026f47
	_FE_READ_SNR                = 0x80026f48
	_FE_READ_BER                = 0x80046f46
	_FE_READ_UNCORRECTED_BLOCKS = 0x80046f49
	_FE_READ_STATUS             = 0x80046f45
)

// Define the BER range
const BER_MIN uint32 = 0
const BER_MAX uint32 = 100000

// FrontendStatus represents the status of the frontend device
type FrontendStatus uint32

// Define the frontend status flags
const (
	FE_HAS_SIGNAL  FrontendStatus = 0x01
	FE_HAS_CARRIER FrontendStatus = 0x02
	FE_HAS_VITERBI FrontendStatus = 0x04
	FE_HAS_SYNC    FrontendStatus = 0x08
	FE_HAS_LOCK    FrontendStatus = 0x10
	FE_TIMEDOUT    FrontendStatus = 0x20
	FE_REINIT      FrontendStatus = 0x40
)

// OpenFrontend opens the frontend device for reading
func OpenFrontend(adapter int) (*os.File, error) {
	frontendPath := fmt.Sprintf("/dev/dvb/adapter%d/frontend0", adapter)
	return os.Open(frontendPath)
}

// ioctl sends an ioctl request to the file descriptor
func ioctl(fd uintptr, request, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// GetStats reads the stats from the frontend device and returns the values based on the specified ValueType
func GetStats(file *os.File) (FrontendStatus, uint16, uint16, uint32, error) {
	fd := file.Fd()

	var signalStrength uint16
	var snr uint16
	var ber uint32
	var status FrontendStatus

	if err := ioctl(fd, _FE_READ_STATUS, uintptr(unsafe.Pointer(&status))); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := ioctl(fd, _FE_READ_SIGNAL_STRENGTH, uintptr(unsafe.Pointer(&signalStrength))); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := ioctl(fd, _FE_READ_SNR, uintptr(unsafe.Pointer(&snr))); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := ioctl(fd, _FE_READ_BER, uintptr(unsafe.Pointer(&ber))); err != nil {
		return 0, 0, 0, 0, err
	}

	signalRelative := uint16(float64(signalStrength) * 100 / 0xffff)
	snrRelative := uint16(100 - (float64(snr) * 100 / 0xffff))

	if ber > BER_MAX {
		ber = BER_MAX
	}
	berScaled := uint32(float64(ber-BER_MIN) * 100 / float64(BER_MAX-BER_MIN))

	return status, signalRelative, snrRelative, berScaled, nil
}

// statusString converts the status flags to a human-readable string
func StatusString(status FrontendStatus) string {
	statuses := []string{}
	if status&FE_HAS_SIGNAL != 0 {
		statuses = append(statuses, "FE_HAS_SIGNAL")
	}
	if status&FE_HAS_CARRIER != 0 {
		statuses = append(statuses, "FE_HAS_CARRIER")
	}
	if status&FE_HAS_VITERBI != 0 {
		statuses = append(statuses, "FE_HAS_VITERBI")
	}
	if status&FE_HAS_SYNC != 0 {
		statuses = append(statuses, "FE_HAS_SYNC")
	}
	if status&FE_HAS_LOCK != 0 {
		statuses = append(statuses, "FE_HAS_LOCK")
	}
	if status&FE_TIMEDOUT != 0 {
		statuses = append(statuses, "FE_TIMEDOUT")
	}
	if status&FE_REINIT != 0 {
		statuses = append(statuses, "FE_REINIT")
	}
	return fmt.Sprintf("%v", statuses)
}
