package intent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/home-ai-union/homeocto/pkg/homeocto/data"
)

// deviceID generates a simple time-based ID for a new device.
func deviceID() string {
	return fmt.Sprintf("device-%d", time.Now().UnixNano())
}

// DeviceMgmtIntent handles device management intents (device.add, device.scan,
// device.remove, device.rename, device.move, device.query.status).
type DeviceMgmtIntent struct {
	deviceStore data.DeviceStore
	spaceStore  data.SpaceStore
}

// NewDeviceMgmtIntent creates a DeviceMgmtIntent backed by the given stores.
// If either store is nil the handler falls through to the large model.
func NewDeviceMgmtIntent(deviceStore data.DeviceStore, spaceStore data.SpaceStore) *DeviceMgmtIntent {
	return &DeviceMgmtIntent{deviceStore: deviceStore, spaceStore: spaceStore}
}

// Types implements Intent.
func (d *DeviceMgmtIntent) Types() []IntentType {
	return []IntentType{
		IntentDeviceAdd,
		IntentDeviceScan,
		IntentDeviceRemove,
		IntentDeviceRename,
		IntentDeviceMove,
		IntentDeviceQueryStatus,
	}
}

// Run executes the device management operation and returns a direct reply.
func (d *DeviceMgmtIntent) Run(_ context.Context, ictx IntentContext) IntentResponse {
	if d.deviceStore == nil {
		return IntentResponse{Handled: false}
	}

	switch ictx.Result.Type {
	case IntentDeviceAdd:
		return d.handleAdd(ictx)
	case IntentDeviceScan:
		return d.handleScan()
	case IntentDeviceRemove:
		return d.handleRemove(ictx)
	case IntentDeviceRename:
		return d.handleRename(ictx)
	case IntentDeviceMove:
		return d.handleMove(ictx)
	case IntentDeviceQueryStatus:
		return d.handleQueryStatus(ictx)
	default:
		return IntentResponse{Handled: false}
	}
}

func (d *DeviceMgmtIntent) handleAdd(ictx IntentContext) IntentResponse {
	name := entityString(ictx.Result.Entities, "device_name")
	if name == "" {
		return IntentResponse{Handled: false}
	}

	spaceName := entityString(ictx.Result.Entities, "space_name")
	if spaceName != "" && d.spaceStore != nil {
		// Verify space exists
		spaces, err := d.spaceStore.GetAll()
		if err == nil {
			found := false
			for _, sp := range spaces {
				if strings.EqualFold(sp.Name, spaceName) {
					spaceName = sp.Name // Use canonical name
					found = true
					break
				}
			}
			if !found {
				spaceName = "" // Space not found
			}
		}
	}

	device := data.Device{
		FromID:    deviceID(),
		Name:      name,
		SpaceName: spaceName,
	}
	if err := d.deviceStore.Save(device); err != nil {
		return errResponse(fmt.Sprintf("����豸��%s��ʧ�ܣ�%s", name, err.Error()), err)
	}
	msg := fmt.Sprintf("������豸��%s����", name)
	if entityString(ictx.Result.Entities, "space_name") != "" && spaceName == "" {
		msg += fmt.Sprintf("��δ�ҵ��ռ䡸%s�����豸δ���䷿�䣩", entityString(ictx.Result.Entities, "space_name"))
	}
	return IntentResponse{Handled: true, Response: msg}
}

func (d *DeviceMgmtIntent) handleScan() IntentResponse {
	// Actual network scanning requires hardware integration; report back to the
	// large model so it can guide the user through the pairing wizard.
	return IntentResponse{Handled: false}
}

func (d *DeviceMgmtIntent) handleRemove(ictx IntentContext) IntentResponse {
	name := entityString(ictx.Result.Entities, "device_name")
	if name == "" {
		return IntentResponse{Handled: false}
	}

	devices, err := d.deviceStore.GetAll()
	if err != nil {
		return errResponse(fmt.Sprintf("��ѯ�豸ʧ�ܣ�%s", err.Error()), err)
	}
	for _, dev := range devices {
		if strings.EqualFold(dev.Name, name) {
			if err := d.deviceStore.Delete(dev.FromID, dev.From); err != nil {
				return errResponse(fmt.Sprintf("ɾ���豸��%s��ʧ�ܣ�%s", name, err.Error()), err)
			}
			return IntentResponse{Handled: true, Response: fmt.Sprintf("��ɾ���豸��%s����", name)}
		}
	}
	return IntentResponse{Handled: true, Response: fmt.Sprintf("δ�ҵ��豸��%s����", name)}
}

func (d *DeviceMgmtIntent) handleRename(ictx IntentContext) IntentResponse {
	oldName := entityString(ictx.Result.Entities, "device_name")
	newName := entityString(ictx.Result.Entities, "new_name")
	if oldName == "" || newName == "" {
		return IntentResponse{Handled: false}
	}

	devices, err := d.deviceStore.GetAll()
	if err != nil {
		return errResponse(fmt.Sprintf("��ѯ�豸ʧ�ܣ�%s", err.Error()), err)
	}
	for _, dev := range devices {
		if strings.EqualFold(dev.Name, oldName) {
			dev.Name = newName
			if err := d.deviceStore.Save(dev); err != nil {
				return errResponse(fmt.Sprintf("�������豸ʧ�ܣ�%s", err.Error()), err)
			}
			return IntentResponse{Handled: true, Response: fmt.Sprintf("�ѽ��豸��%s��������Ϊ��%s����", oldName, newName)}
		}
	}
	return IntentResponse{Handled: true, Response: fmt.Sprintf("δ�ҵ��豸��%s����", oldName)}
}

func (d *DeviceMgmtIntent) handleMove(ictx IntentContext) IntentResponse {
	deviceName := entityString(ictx.Result.Entities, "device_name")
	spaceName := entityString(ictx.Result.Entities, "space_name")
	if deviceName == "" || spaceName == "" {
		return IntentResponse{Handled: false}
	}

	// Resolve space.
	var resolvedSpaceName string
	if d.spaceStore != nil {
		spaces, err := d.spaceStore.GetAll()
		if err != nil {
			return errResponse(fmt.Sprintf("��ѯ�ռ�ʧ�ܣ�%s", err.Error()), err)
		}
		for _, sp := range spaces {
			if strings.EqualFold(sp.Name, spaceName) {
				resolvedSpaceName = sp.Name
				break
			}
		}
		if resolvedSpaceName == "" {
			return IntentResponse{Handled: true, Response: fmt.Sprintf("δ�ҵ��ռ䡸%s����", spaceName)}
		}
	}

	devices, err := d.deviceStore.GetAll()
	if err != nil {
		return errResponse(fmt.Sprintf("��ѯ�豸ʧ�ܣ�%s", err.Error()), err)
	}
	for _, dev := range devices {
		if strings.EqualFold(dev.Name, deviceName) {
			dev.SpaceName = resolvedSpaceName
			if err := d.deviceStore.Save(dev); err != nil {
				return errResponse(fmt.Sprintf("�ƶ��豸ʧ�ܣ�%s", err.Error()), err)
			}
			return IntentResponse{Handled: true, Response: fmt.Sprintf("�ѽ��豸��%s���ƶ�����%s����", deviceName, spaceName)}
		}
	}
	return IntentResponse{Handled: true, Response: fmt.Sprintf("δ�ҵ��豸��%s����", deviceName)}
}

func (d *DeviceMgmtIntent) handleQueryStatus(ictx IntentContext) IntentResponse {
	name := entityString(ictx.Result.Entities, "device_name")

	// Query a specific device.
	if name != "" {
		devices, err := d.deviceStore.GetAll()
		if err != nil {
			return errResponse(fmt.Sprintf("��ѯ�豸ʧ�ܣ�%s", err.Error()), err)
		}
		for _, dev := range devices {
			if strings.EqualFold(dev.Name, name) {
				return IntentResponse{
					Handled:  true,
					Response: formatDeviceStatus(dev),
				}
			}
		}
		return IntentResponse{Handled: true, Response: fmt.Sprintf("δ�ҵ��豸��%s����", name)}
	}

	// Query all devices.
	devices, err := d.deviceStore.GetAll()
	if err != nil {
		return errResponse(fmt.Sprintf("��ѯ�豸�б�ʧ�ܣ�%s", err.Error()), err)
	}
	if len(devices) == 0 {
		return IntentResponse{Handled: true, Response: "��ǰû���κ��豸��"}
	}
	lines := make([]string, 0, len(devices))
	for _, dev := range devices {
		lines = append(lines, fmt.Sprintf("��%s����%s��", dev.Name, dev.From))
	}
	return IntentResponse{
		Handled:  true,
		Response: fmt.Sprintf("���� %d ̨�豸��%s��", len(devices), strings.Join(lines, "��")),
	}
}

func formatDeviceStatus(dev data.Device) string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "�豸��%s��", dev.Name)
	if dev.From != "" {
		fmt.Fprintf(sb, "��%s��", dev.From)
	}
	sb.WriteString("��״̬δ֪��")
	return sb.String()
}
