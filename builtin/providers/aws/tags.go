package aws

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	spotRequestIDPrefix        = "sir-"
	spotRequestSpotInstance_ID = "spot_instance_id"
)

// tagsSchema returns the schema to use for tags.
//
func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
	}
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTags(conn *ec2.EC2, d *schema.ResourceData) error {
	var err error

	id := d.Id()

	// Check for a Spot Instance Requests as it requires special
	// handeling, the tagging process will tag the spot request it self
	// as its id is stored in d.Id(), to tag the instance the spot
	// request creates we need to use the the instance id which is stored
	// under key 'spot_instance_id'
	if strings.Contains(id, spotRequestIDPrefix) {
		spotInstID := d.Get(spotRequestSpotInstance_ID).(string)
		err = setTagsActual(conn, d, spotInstID)
		if err != nil {
			return err
		}
	}

	err = setTagsActual(conn, d, id)

	if err != nil {
		return err
	}

	return nil
}

func setTagsActual(conn *ec2.EC2, d *schema.ResourceData, id string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTags(tagsFromMap(o), tagsFromMap(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v from %s", remove, id)
			_, err := conn.DeleteTags(&ec2.DeleteTagsInput{
				Resources: []*string{aws.String(id)},
				Tags:      remove,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s for %s", create, id)
			_, err := conn.CreateTags(&ec2.CreateTagsInput{
				Resources: []*string{aws.String(id)},
				Tags:      create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTags(oldTags, newTags []*ec2.Tag) ([]*ec2.Tag, []*ec2.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*ec2.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMap(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMap(m map[string]interface{}) []*ec2.Tag {
	result := make([]*ec2.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMap(ts []*ec2.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
