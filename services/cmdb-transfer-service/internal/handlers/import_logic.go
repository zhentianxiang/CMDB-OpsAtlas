package handlers

import (
	"cmdb-v2/pkg/models"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const previewItemLimit = 20

type importSummary struct {
	Clusters     int `json:"clusters"`
	Hosts        int `json:"hosts"`
	Apps         int `json:"apps"`
	Ports        int `json:"ports"`
	Domains      int `json:"domains"`
	Dependencies int `json:"dependencies"`
}

type previewResourceDiff struct {
	Resource  string   `json:"resource"`
	AddCount  int      `json:"add_count"`
	SkipCount int      `json:"skip_count"`
	AddItems  []string `json:"add_items"`
	SkipItems []string `json:"skip_items"`
}

type previewSection struct {
	Summary   importSummary         `json:"summary"`
	Resources []previewResourceDiff `json:"resources"`
}

type previewResponse struct {
	Append previewSection `json:"append"`
	Overwrite struct {
		Current  importSummary `json:"current"`
		Incoming importSummary `json:"incoming"`
	} `json:"overwrite"`
}

type importExecutionResult struct {
	Mode    string        `json:"mode"`
	Added   importSummary `json:"added"`
	Skipped importSummary `json:"skipped"`
}

type resourceDiffCounter struct {
	Resource  string
	AddCount  int
	SkipCount int
	AddItems  []string
	SkipItems []string
}

type importContext struct {
	persist bool
	tx      *gorm.DB
	userID  uint

	nextClusterID uint
	nextHostID    uint
	nextAppID     uint
	nextPortID    uint
	nextDomainID  uint
	nextDepID     uint

	clusterByKey    map[string]*models.Cluster
	hostByKey       map[string]*models.Host
	appByKey        map[string]*models.App
	portByKey       map[string]*models.Port
	domainByKey     map[string]*models.Domain
	dependencyByKey map[string]*models.Dependency

	clusterIDMap map[uint]uint
	hostIDMap    map[uint]uint
	appIDMap     map[uint]uint
        domainIDMap  map[uint]uint

	added   importSummary
	skipped importSummary
	diffs   map[string]*resourceDiffCounter
}

func (h *Handler) PreviewCMDB(uID uint, role string, payload *exportPayload) (*previewResponse, error) {
	current, err := h.loadCurrentImportState(h.DB, uID, role)
	if err != nil {
		return nil, err
	}

	ctx := newImportContext(uID, current, false, nil)
	if err := ctx.processPayload(payload); err != nil {
		return nil, err
	}

	resp := &previewResponse{
		Append: previewSection{
			Summary:   ctx.added,
			Resources: ctx.exportDiffs(),
		},
	}
	resp.Overwrite.Current = current.summary()
	resp.Overwrite.Incoming = payload.summary()
	return resp, nil
}

func (h *Handler) AppendImportCMDB(uID uint, role string, payload *exportPayload) (*importExecutionResult, error) {
	result := &importExecutionResult{Mode: "append"}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		current, err := h.loadCurrentImportState(tx, uID, role)
		if err != nil {
			return err
		}
		ctx := newImportContext(uID, current, true, tx)
		if err := ctx.processPayload(payload); err != nil {
			return err
		}
		result.Added = ctx.added
		result.Skipped = ctx.skipped
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type currentImportState struct {
	clusters     []models.Cluster
	hosts        []models.Host
	apps         []models.App
	ports        []models.Port
	domains      []models.Domain
	dependencies []models.Dependency
}

func (s currentImportState) summary() importSummary {
	return importSummary{
		Clusters:     len(s.clusters),
		Hosts:        len(s.hosts),
		Apps:         len(s.apps),
		Ports:        len(s.ports),
		Domains:      len(s.domains),
		Dependencies: len(s.dependencies),
	}
}

func (p exportPayload) summary() importSummary {
	return importSummary{
		Clusters:     len(p.Clusters),
		Hosts:        len(p.Hosts),
		Apps:         len(p.Apps),
		Ports:        len(p.Ports),
		Domains:      len(p.Domains),
		Dependencies: len(p.Dependencies),
	}
}

func (h *Handler) loadCurrentImportState(db *gorm.DB, uID uint, role string) (*currentImportState, error) {
	state := &currentImportState{}
	
	filter := func(q *gorm.DB) *gorm.DB {
		if role == "admin" {
			return q
		}
		return q.Where("user_id = ?", uID)
	}

	if err := filter(db.Order("id ASC")).Find(&state.clusters).Error; err != nil {
		return nil, err
	}
	if err := filter(db.Order("id ASC")).Find(&state.hosts).Error; err != nil {
		return nil, err
	}
	if err := filter(db.Order("id ASC")).Find(&state.apps).Error; err != nil {
		return nil, err
	}
	if err := filter(db.Order("id ASC")).Find(&state.ports).Error; err != nil {
		return nil, err
	}
	if err := filter(db.Order("id ASC")).Find(&state.domains).Error; err != nil {
		return nil, err
	}
	if err := filter(db.Order("id ASC")).Find(&state.dependencies).Error; err != nil {
		return nil, err
	}
	return state, nil
}

func newImportContext(uID uint, current *currentImportState, persist bool, tx *gorm.DB) *importContext {
	ctx := &importContext{
		persist:         persist,
		tx:              tx,
		userID:          uID,
		clusterByKey:    map[string]*models.Cluster{},
		hostByKey:       map[string]*models.Host{},
		appByKey:        map[string]*models.App{},
		portByKey:       map[string]*models.Port{},
		domainByKey:     map[string]*models.Domain{},
		dependencyByKey: map[string]*models.Dependency{},
		clusterIDMap:    map[uint]uint{},
		hostIDMap:       map[uint]uint{},
		appIDMap:        map[uint]uint{},
                domainIDMap:     map[uint]uint{},
		diffs:           map[string]*resourceDiffCounter{},
	}

	for i := range current.clusters {
		item := current.clusters[i]
		ctx.clusterByKey[clusterKey(item.Name)] = &item
		if item.ID >= ctx.nextClusterID {
			ctx.nextClusterID = item.ID + 1
		}
	}
	for i := range current.hosts {
		item := current.hosts[i]
		ctx.addHostLookup(&item)
		if item.ID >= ctx.nextHostID {
			ctx.nextHostID = item.ID + 1
		}
	}
	for i := range current.apps {
		item := current.apps[i]
		ctx.appByKey[appKey(item.HostID, item.Name)] = &item
		if item.ID >= ctx.nextAppID {
			ctx.nextAppID = item.ID + 1
		}
	}
	for i := range current.ports {
		item := current.ports[i]
		ctx.portByKey[portKey(item.AppID, item.Port, item.Protocol)] = &item
		if item.ID >= ctx.nextPortID {
			ctx.nextPortID = item.ID + 1
		}
	}
	for i := range current.domains {
		item := current.domains[i]
		ctx.domainByKey[domainKey(item.Domain)] = &item
		if item.ID >= ctx.nextDomainID {
			ctx.nextDomainID = item.ID + 1
		}
	}
	for i := range current.dependencies {
		item := current.dependencies[i]
		ctx.dependencyByKey[dependencyKey(item)] = &item
		if item.ID >= ctx.nextDepID {
			ctx.nextDepID = item.ID + 1
		}
	}

	if ctx.nextClusterID == 0 {
		ctx.nextClusterID = 1
	}
	if ctx.nextHostID == 0 {
		ctx.nextHostID = 1
	}
	if ctx.nextAppID == 0 {
		ctx.nextAppID = 1
	}
	if ctx.nextPortID == 0 {
		ctx.nextPortID = 1
	}
	if ctx.nextDomainID == 0 {
		ctx.nextDomainID = 1
	}
	if ctx.nextDepID == 0 {
		ctx.nextDepID = 1
	}

	return ctx
}

func (ctx *importContext) processPayload(payload *exportPayload) error {
	for _, item := range payload.Clusters {
		if _, err := ctx.ensureCluster(item); err != nil {
			return err
		}
	}
	for _, item := range payload.Hosts {
		if _, err := ctx.ensureHost(item); err != nil {
			return err
		}
	}
	for _, item := range payload.Apps {
		if _, err := ctx.ensureApp(item); err != nil {
			return err
		}
	}
	for _, item := range payload.Ports {
		if err := ctx.ensurePort(item); err != nil {
			return err
		}
	}
	for _, item := range payload.Domains {
		if err := ctx.ensureDomain(item); err != nil {
			return err
		}
	}
	for _, item := range payload.Dependencies {
		if err := ctx.ensureDependency(item); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *importContext) ensureCluster(src models.Cluster) (uint, error) {
	key := clusterKey(src.Name)
	if current, ok := ctx.clusterByKey[key]; ok {
		ctx.mapImportedClusterID(src.ID, current.ID)
		ctx.record("clusters", "skip", clusterLabel(src))
		return current.ID, nil
	}

	item := models.Cluster{
		UserID: ctx.userID,
		Name: src.Name, 
		Type: src.Type, 
		Env: src.Env, 
		Remark: src.Remark,
	}
	if err := ctx.createModel("clusters", &item, clusterLabel(src)); err != nil {
		return 0, err
	}
	ctx.clusterByKey[key] = &item
	ctx.mapImportedClusterID(src.ID, item.ID)
	return item.ID, nil
}

func (ctx *importContext) ensureHost(src models.Host) (uint, error) {
	if current := ctx.findHost(src); current != nil {
		ctx.mapImportedHostID(src.ID, current.ID)
		ctx.record("hosts", "skip", hostLabel(src))
		return current.ID, nil
	}

	item := models.Host{
		UserID:    ctx.userID,
		Name:      src.Name,
		IP:        src.IP,
		PublicIP:  src.PublicIP,
		PrivateIP: src.PrivateIP,
		CPU:       src.CPU,
		Memory:    src.Memory,
		OS:        src.OS,
		Status:    src.Status,
		Remark:    src.Remark,
	}
	if src.ClusterID != nil {
		clusterID, ok := ctx.clusterIDMap[*src.ClusterID]
		if !ok {
			return 0, fmt.Errorf("追加导入失败: 主机 %s 关联集群不存在", hostLabel(src))
		}
		item.ClusterID = uintPtr(clusterID)
	}
	if err := ctx.createModel("hosts", &item, hostLabel(src)); err != nil {
		return 0, err
	}
	ctx.addHostLookup(&item)
	ctx.mapImportedHostID(src.ID, item.ID)
	return item.ID, nil
}

func (ctx *importContext) ensureApp(src models.App) (uint, error) {
	hostID, ok := ctx.hostIDMap[src.HostID]
	if !ok {
		return 0, fmt.Errorf("追加导入失败: 应用 %s 关联主机不存在", appLabel(src))
	}
	key := appKey(hostID, src.Name)
	if current, ok := ctx.appByKey[key]; ok {
		ctx.mapImportedAppID(src.ID, current.ID)
		ctx.record("apps", "skip", appLabelWithHostID(src.Name, hostID))
		return current.ID, nil
	}

	item := models.App{
		UserID:     ctx.userID,
		Name:       src.Name,
		HostID:     hostID,
		Type:       src.Type,
		Version:    src.Version,
		DeployType: src.DeployType,
		Remark:     src.Remark,
	}
	if err := ctx.createModel("apps", &item, appLabelWithHostID(src.Name, hostID)); err != nil {
		return 0, err
	}
	ctx.appByKey[key] = &item
	ctx.mapImportedAppID(src.ID, item.ID)
	return item.ID, nil
}

func (ctx *importContext) ensurePort(src models.Port) error {
	appID, ok := ctx.appIDMap[src.AppID]
	if !ok {
		return fmt.Errorf("追加导入失败: 端口 %s 关联应用不存在", portLabel(src.Port, src.Protocol))
	}
	key := portKey(appID, src.Port, src.Protocol)
	if _, ok := ctx.portByKey[key]; ok {
		ctx.record("ports", "skip", portLabel(src.Port, src.Protocol))
		return nil
	}

	item := models.Port{
		UserID:   ctx.userID,
		AppID:    appID, 
		Port:     src.Port, 
		Protocol: src.Protocol, 
		IsPublic: src.IsPublic, 
		Remark:   src.Remark,
	}
	if err := ctx.createModel("ports", &item, portLabel(src.Port, src.Protocol)); err != nil {
		return err
	}
	ctx.portByKey[key] = &item
	return nil
}

func (ctx *importContext) ensureDomain(src models.Domain) error {
	key := domainKey(src.Domain)
	if old, ok := ctx.domainByKey[key]; ok {
                ctx.domainIDMap[src.ID] = old.ID
		ctx.record("domains", "skip", domainLabel(src.Domain))
		return nil
	}

	item := models.Domain{
		UserID: ctx.userID,
		Domain: src.Domain, 
		Remark: src.Remark,
	}
	if src.AppID != nil {
		appID, ok := ctx.appIDMap[*src.AppID]
		if !ok {
			return fmt.Errorf("追加导入失败: 域名 %s 关联应用不存在", domainLabel(src.Domain))
		}
		item.AppID = uintPtr(appID)
	}
	if src.HostID != nil {
		hostID, ok := ctx.hostIDMap[*src.HostID]
		if !ok {
			return fmt.Errorf("追加导入失败: 域名 %s 关联主机不存在", domainLabel(src.Domain))
		}
		item.HostID = uintPtr(hostID)
	}
	if err := ctx.createModel("domains", &item, domainLabel(src.Domain)); err != nil {
                return err
	}
        ctx.domainIDMap[src.ID] = item.ID
        ctx.domainByKey[key] = &item
        return nil
}


func (ctx *importContext) ensureDependency(src models.Dependency) error {
	item := models.Dependency{
		UserID: ctx.userID,
		SourceNode: src.SourceNode, 
		TargetNode: src.TargetNode, 
		Desc: src.Desc, 
		Remark: src.Remark,
	}
	if src.SourceAppID != nil {
		appID, ok := ctx.appIDMap[*src.SourceAppID]
		if !ok {
			return fmt.Errorf("追加导入失败: 依赖 %s 的调用方应用不存在", dependencyLabel(src))
		}
		item.SourceAppID = uintPtr(appID)
	}
	if src.TargetAppID != nil {
		appID, ok := ctx.appIDMap[*src.TargetAppID]
		if !ok {
			return fmt.Errorf("追加导入失败: 依赖 %s 的被调用应用不存在", dependencyLabel(src))
		}
		item.TargetAppID = uintPtr(appID)
	}
	if src.SourceHostID != nil {
		hostID, ok := ctx.hostIDMap[*src.SourceHostID]
		if !ok {
			return fmt.Errorf("追加导入失败: 依赖 %s 的调用方主机不存在", dependencyLabel(src))
		}
		item.SourceHostID = uintPtr(hostID)
	}
	if src.TargetHostID != nil {
		hostID, ok := ctx.hostIDMap[*src.TargetHostID]
		if !ok {
			return fmt.Errorf("追加导入失败: 依赖 %s 的被调用方主机不存在", dependencyLabel(src))
		}
		item.TargetHostID = uintPtr(hostID)
	}
	if src.DomainID != nil {
		domainID, ok := ctx.domainIDMap[*src.DomainID]
		if ok {
			item.DomainID = uintPtr(domainID)
		} else {
			item.DomainID = nil
		}
	}

	key := dependencyKey(item)
	if _, ok := ctx.dependencyByKey[key]; ok {
		ctx.record("dependencies", "skip", dependencyLabel(item))
		return nil
	}

	if err := ctx.createModel("dependencies", &item, dependencyLabel(item)); err != nil {
		return err
	}
        ctx.dependencyByKey[key] = &item
        return nil
} 
func (ctx *importContext) createModel(resource string, item interface{}, label string) error {
	switch typed := item.(type) {
	case *models.Cluster:
		if ctx.persist {
			if err := ctx.tx.Create(typed).Error; err != nil {
				return err
			}
		} else {
			typed.ID = ctx.nextClusterID
			ctx.nextClusterID++
		}
	case *models.Host:
		if ctx.persist {
			if err := ctx.tx.Create(typed).Error; err != nil {
				return err
			}
		} else {
			typed.ID = ctx.nextHostID
			ctx.nextHostID++
		}
	case *models.App:
		if ctx.persist {
			if err := ctx.tx.Create(typed).Error; err != nil {
				return err
			}
		} else {
			typed.ID = ctx.nextAppID
			ctx.nextAppID++
		}
	case *models.Port:
		if ctx.persist {
			if err := ctx.tx.Create(typed).Error; err != nil {
				return err
			}
		} else {
			typed.ID = ctx.nextPortID
			ctx.nextPortID++
		}
	case *models.Domain:
		if ctx.persist {
			if err := ctx.tx.Create(typed).Error; err != nil {
				return err
			}
		} else {
			typed.ID = ctx.nextDomainID
			ctx.nextDomainID++
		}
	case *models.Dependency:
		if ctx.persist {
			if err := ctx.tx.Create(typed).Error; err != nil {
				return err
			}
		} else {
			typed.ID = ctx.nextDepID
			ctx.nextDepID++
		}
	}

	ctx.record(resource, "add", label)
	return nil
}

func (ctx *importContext) record(resource string, action string, label string) {
	diff := ctx.diffs[resource]
	if diff == nil {
		diff = &resourceDiffCounter{Resource: resource}
		ctx.diffs[resource] = diff
	}

	switch resource {
	case "clusters":
		if action == "add" {
			ctx.added.Clusters++
		} else {
			ctx.skipped.Clusters++
		}
	case "hosts":
		if action == "add" {
			ctx.added.Hosts++
		} else {
			ctx.skipped.Hosts++
		}
	case "apps":
		if action == "add" {
			ctx.added.Apps++
		} else {
			ctx.skipped.Apps++
		}
	case "ports":
		if action == "add" {
			ctx.added.Ports++
		} else {
			ctx.skipped.Ports++
		}
	case "domains":
		if action == "add" {
			ctx.added.Domains++
		} else {
			ctx.skipped.Domains++
		}
	case "dependencies":
		if action == "add" {
			ctx.added.Dependencies++
		} else {
			ctx.skipped.Dependencies++
		}
	}

	if action == "add" {
		diff.AddCount++
		if len(diff.AddItems) < previewItemLimit {
			diff.AddItems = append(diff.AddItems, label)
		}
		return
	}

	diff.SkipCount++
	if len(diff.SkipItems) < previewItemLimit {
		diff.SkipItems = append(diff.SkipItems, label)
	}
}

func (ctx *importContext) exportDiffs() []previewResourceDiff {
	order := []string{"clusters", "hosts", "apps", "ports", "domains", "dependencies"}
	items := make([]previewResourceDiff, 0, len(order))
	for _, key := range order {
		diff := ctx.diffs[key]
		if diff == nil {
			diff = &resourceDiffCounter{Resource: key}
		}
		items = append(items, previewResourceDiff{
			Resource:  diff.Resource,
			AddCount:  diff.AddCount,
			SkipCount: diff.SkipCount,
			AddItems:  diff.AddItems,
			SkipItems: diff.SkipItems,
		})
	}
	return items
}

func (ctx *importContext) mapImportedClusterID(sourceID uint, actualID uint) {
	if sourceID > 0 {
		ctx.clusterIDMap[sourceID] = actualID
	}
}

func (ctx *importContext) mapImportedHostID(sourceID uint, actualID uint) {
	if sourceID > 0 {
		ctx.hostIDMap[sourceID] = actualID
	}
}

func (ctx *importContext) mapImportedAppID(sourceID uint, actualID uint) {
	if sourceID > 0 {
		ctx.appIDMap[sourceID] = actualID
	}
}

func (ctx *importContext) addHostLookup(item *models.Host) {
	for _, key := range hostKeys(*item) {
		ctx.hostByKey[key] = item
	}
}

func (ctx *importContext) findHost(src models.Host) *models.Host {
	for _, key := range hostKeys(src) {
		if item, ok := ctx.hostByKey[key]; ok {
			return item
		}
	}
	return nil
}

func clusterKey(name string) string { return normalize(name) }

func hostKeys(item models.Host) []string {
	keys := make([]string, 0, 4)
	appendKey := func(prefix string, value string) {
		value = normalize(value)
		if value == "" {
			return
		}
		keys = append(keys, prefix+":"+value)
	}
	appendKey("private", item.PrivateIP)
	appendKey("public", item.PublicIP)
	appendKey("ip", item.IP)
	appendKey("name", item.Name)
	return keys
}

func appKey(hostID uint, name string) string {
	return fmt.Sprintf("%d|%s", hostID, normalize(name))
}

func portKey(appID uint, port int, protocol string) string {
	return fmt.Sprintf("%d|%d|%s", appID, port, normalize(protocol))
}

func domainKey(domain string) string { return normalize(domain) }

func dependencyKey(item models.Dependency) string {
	return fmt.Sprintf(
		"sa:%d|ta:%d|sh:%d|th:%d|sn:%s|tn:%s|d:%s",
		uintValue(item.SourceAppID),
		uintValue(item.TargetAppID),
		uintValue(item.SourceHostID),
		uintValue(item.TargetHostID),
		normalize(item.SourceNode),
		normalize(item.TargetNode),
		normalize(item.Desc),
	)
}

func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func uintValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func uintPtr(value uint) *uint { return &value }

func clusterLabel(item models.Cluster) string { return strings.TrimSpace(item.Name) }

func hostLabel(item models.Host) string {
	ip := strings.TrimSpace(item.PrivateIP)
	if ip == "" {
		ip = strings.TrimSpace(item.PublicIP)
	}
	if ip == "" {
		ip = strings.TrimSpace(item.IP)
	}
	if ip == "" {
		return strings.TrimSpace(item.Name)
	}
	if item.Name == "" {
		return ip
	}
	return fmt.Sprintf("%s (%s)", strings.TrimSpace(item.Name), ip)
}

func appLabel(item models.App) string { return appLabelWithHostID(item.Name, item.HostID) }

func appLabelWithHostID(name string, hostID uint) string {
	label := strings.TrimSpace(name)
	if hostID == 0 {
		return label
	}
	return fmt.Sprintf("%s [host:%d]", label, hostID)
}

func portLabel(port int, protocol string) string {
	protocol = strings.ToUpper(strings.TrimSpace(protocol))
	if protocol == "" {
		protocol = "TCP"
	}
	return fmt.Sprintf("%d/%s", port, protocol)
}

func domainLabel(domain string) string { return strings.TrimSpace(domain) }

func dependencyLabel(item models.Dependency) string {
	return fmt.Sprintf(
		"SA:%d TA:%d SH:%d TH:%d %s -> %s",
		uintValue(item.SourceAppID),
		uintValue(item.TargetAppID),
		uintValue(item.SourceHostID),
		uintValue(item.TargetHostID),
		strings.TrimSpace(item.SourceNode),
		strings.TrimSpace(item.TargetNode),
	)
}
