package mackerel

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/mackerelio/mackerel-client-go"
)

var serviceResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
	},
}

var monitorResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"skip_default": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
	},
}

func resourceMackerelNotificationGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceMackerelNotificationGroupCreate,
		Read:   resourceMackerelNotificationGroupRead,
		Update: resourceMackerelNotificationGroupUpdate,
		Delete: resourceMackerelNotificationGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"notification_level": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"all", "critical"}, false),
				Default:      "all",
			},
			"child_notification_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"child_channel_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"monitor": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     monitorResource,
			},
			"service": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     serviceResource,
			},
		},
	}
}

func resourceMackerelNotificationGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mackerel.Client)
	group, err := client.CreateNotificationGroup(buildNotificationGroupStruct(d))
	if err != nil {
		return err
	}
	d.SetId(group.ID)

	return resourceMackerelNotificationGroupRead(d, meta)
}

func resourceMackerelNotificationGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mackerel.Client)
	groups, err := client.FindNotificationGroups()
	if err != nil {
		return err
	}

	for _, group := range groups {
		if group.ID == d.Id() {
			d.Set("name", group.Name)
			d.Set("notification_level", group.NotificationLevel)
			d.Set("child_notification_group_ids", flattenStringSet(group.ChildNotificationGroupIDs))
			d.Set("child_channel_ids", flattenStringSet(group.ChildChannelIDs))
			d.Set("monitor", flattenMonitors(group.Monitors))
			d.Set("service", flattenServices(group.Services))
			break
		}
	}

	return nil
}

func resourceMackerelNotificationGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mackerel.Client)
	group, err := client.UpdateNotificationGroup(d.Id(), buildNotificationGroupStruct(d))
	if err != nil {
		return err
	}
	d.SetId(group.ID)

	return resourceMackerelNotificationGroupRead(d, meta)
}

func resourceMackerelNotificationGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mackerel.Client)
	_, err := client.DeleteNotificationGroup(d.Id())
	return err
}

func buildNotificationGroupStruct(d *schema.ResourceData) *mackerel.NotificationGroup {
	group := &mackerel.NotificationGroup{
		Name:                      d.Get("name").(string),
		ChildNotificationGroupIDs: expandStringList(d.Get("child_notification_group_ids").(*schema.Set).List()),
		ChildChannelIDs:           expandStringList(d.Get("child_channel_ids").(*schema.Set).List()),
		Monitors:                  expandMonitorSet(d.Get("monitor").(*schema.Set)),
		Services:                  expandServiceSet(d.Get("service").(*schema.Set)),
	}

	switch d.Get("notification_level").(string) {
	case "all":
		group.NotificationLevel = mackerel.NotificationLevelAll
	case "critical":
		group.NotificationLevel = mackerel.NotificationLevelCritical
	}

	return group
}

func expandMonitorSet(set *schema.Set) []*mackerel.NotificationGroupMonitor {
	monitors := make([]*mackerel.NotificationGroupMonitor, 0, set.Len())

	for _, monitor := range set.List() {
		monitor := monitor.(map[string]interface{})
		monitors = append(monitors, &mackerel.NotificationGroupMonitor{
			ID:          monitor["id"].(string),
			SkipDefault: monitor["skip_default"].(bool),
		})
	}

	return monitors
}

func flattenMonitors(v []*mackerel.NotificationGroupMonitor) *schema.Set {
	monitors := make([]interface{}, 0, len(v))

	for _, monitor := range v {
		monitors = append(monitors, map[string]interface{}{
			"id":           monitor.ID,
			"skip_default": monitor.SkipDefault,
		})
	}

	return schema.NewSet(schema.HashResource(monitorResource), monitors)
}

func expandServiceSet(set *schema.Set) []*mackerel.NotificationGroupService {
	services := make([]*mackerel.NotificationGroupService, 0, set.Len())

	for _, service := range set.List() {
		service := service.(map[string]interface{})
		services = append(services, &mackerel.NotificationGroupService{Name: service["name"].(string)})
	}

	return services
}

func flattenServices(v []*mackerel.NotificationGroupService) *schema.Set {
	services := make([]interface{}, 0, len(v))

	for _, srv := range v {
		services = append(services, map[string]interface{}{
			"name": srv.Name,
		})
	}

	return schema.NewSet(schema.HashResource(serviceResource), services)
}
