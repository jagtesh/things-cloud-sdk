package thingscloud

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// State is created by applying all history items in order.
// Note that the hierarchy within the state (e.g. area > tasks > tasks > check list items)
// is modelled with pointers between the different maps, so concurrent modification
// is not safe.
type State struct {
	Areas          map[string]*Area
	Tasks          map[string]*Task
	Tags           map[string]*Tag
	CheckListItems map[string]*CheckList
}

// NewState creates a new, empty state
func NewState() *State {
	return &State{
		Areas:          map[string]*Area{},
		Tags:           map[string]*Tag{},
		CheckListItems: map[string]*CheckList{},
		Tasks:          map[string]*Task{},
	}
}

// 0|uuid|TEXT|0||1
// 1|logInterval|INTEGER|0||0
// 2|manualLogDate|REAL|0||0
// 3|groupTodayByParent|INTEGER|0||0
type Setting struct{}

// Task describes a Task inside things.
// 0|uuid|TEXT|0||1
// 1|userModificationDate|REAL|0||0
// 2|creationDate|REAL|0||0
// 3|trashed|INTEGER|0||0
// 4|type|INTEGER|0||0
// 5|title|TEXT|0||0
// 6|notes|TEXT|0||0
// 7|dueDate|REAL|0||0
// 8|dueDateOffset|INTEGER|0||0
// 9|status|INTEGER|0||0
// 10|stopDate|REAL|0||0
// 11|start|INTEGER|0||0
// 12|startDate|REAL|0||0
// 13|index|INTEGER|0||0
// 14|todayIndex|INTEGER|0||0
// 15|area|TEXT|0||0
// 16|project|TEXT|0||0
// 17|repeatingTemplate|TEXT|0||0
// 18|delegate|TEXT|0||0
// 19|recurrenceRule|BLOB|0||0
// 20|instanceCreationStartDate|REAL|0||0
// 21|instanceCreationPaused|INTEGER|0||0
// 22|instanceCreationCount|INTEGER|0||0
// 23|afterCompletionReferenceDate|REAL|0||0
// 24|actionGroup|TEXT|0||0
// 25|untrashedLeafActionsCount|INTEGER|0||0
// 26|openUntrashedLeafActionsCount|INTEGER|0||0
// 27|checklistItemsCount|INTEGER|0||0
// 28|openChecklistItemsCount|INTEGER|0||0
// 29|startBucket|INTEGER|0||0
// 30|alarmTimeOffset|REAL|0||0
// 31|lastAlarmInteractionDate|REAL|0||0
// 32|todayIndexReferenceDate|REAL|0||0
// 33|nextInstanceStartDate|REAL|0||0
// 34|dueDateSuppressionDate|REAL|0||0
type Task struct {
	ID               string
	CreationDate     time.Time
	ModificationDate *time.Time
	Status           TaskStatus
	Title            string
	Note             string
	ScheduledDate    *time.Time
	CompletionDate   *time.Time
	DeadlineDate     *time.Time
	Index            int
	AreaIDs          []string
	ParentTaskIDs    []string
	InTrash          bool
	Schedule         TaskSchedule
	IsProject        bool
}

// Subtasks returns tasks grouped together with under a root task
func (s *State) Subtasks(root *Task) []*Task {
	tasks := []*Task{}
	for _, task := range s.Tasks {
		if task.Status == TaskStatusCompleted {
			continue
		}
		if task == root {
			continue
		}
		if task.InTrash {
			continue
		}
		isChild := false
		for _, taskID := range task.ParentTaskIDs {
			isChild = isChild || taskID == root.ID
		}
		if isChild {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Index < tasks[j].Index
	})
	return tasks
}

func hasArea(task *Task, state *State) bool {
	if len(task.AreaIDs) != 0 {
		return true
	}
	if len(task.ParentTaskIDs) == 0 {
		return false
	}
	for _, taskID := range task.ParentTaskIDs {
		if hasArea(state.Tasks[taskID], state) {
			return true
		}
	}
	return false
}

func (s *State) TasksWithoutArea() []*Task {
	tasks := []*Task{}
	for _, task := range s.Tasks {
		if task.Status == TaskStatusCompleted {
			continue
		}
		if len(task.ParentTaskIDs) != 0 {
			continue
		}
		if task.InTrash {
			continue
		}
		if !hasArea(task, s) {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Index < tasks[j].Index
	})
	return tasks
}

// TasksByArea returns tasks associated with a given area
func (s *State) TasksByArea(area *Area) []*Task {
	tasks := []*Task{}
	for _, task := range s.Tasks {
		if task.Status == TaskStatusCompleted {
			continue
		}
		if task.InTrash {
			continue
		}
		isChild := false
		for _, areaID := range task.AreaIDs {
			isChild = isChild || areaID == area.ID
		}
		if isChild {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Index < tasks[j].Index
	})
	return tasks
}

type TaskItemPayload struct {
	Index             *int          `json:"ix,omitempty"`
	CreationDate      *Timestamp    `json:"cd,omitempty"`
	ModificationDate  *Timestamp    `json:"md,omitempty"`
	ScheduledDate     *Timestamp    `json:"sr,omitempty"`
	CompletionDate    *Timestamp    `json:"sp,omitempty"`
	DeadlineDate      *Timestamp    `json:"dd,omitempty"`
	Status            *TaskStatus   `json:"ss,omitempty"`
	IsProject         *Boolean      `json:"tp,omitempty"`
	Title             *string       `json:"tt,omitempty"`
	Note              *string       `json:"nt,omitempty"`
	AreaIDs           *[]string     `json:"ar,omitempty"`
	ParentTaskIDs     *[]string     `json:"pr,omitempty"`
	TagIDs            []string      `json:"tg,omitempty"`
	InTrash           *bool         `json:"tr,omitempty"`
	RecurrenceTaskIDs *[]string     `json:"rt,omitempty"`
	Schedule          *TaskSchedule `json:"st,omitempty"`
	// "rr": {
	//   "fu": 16,
	//   "sr": 1495670400,
	//   "of": [
	//     {
	//       "dy": 0
	//     }
	//   ],
	//   "ts": -3,
	//   "tp": 0,
	//   "fa": 1,
	//   "rc": 2,
	//   "ia": 1495929600,
	//   "rrv": 4
	// },
}

// taskItem describes an event on a task
type TaskItem struct {
	Item
	P TaskItemPayload `json:"p"`
}

func (t TaskItem) ID() string {
	return t.Item.ID
}

func (s *State) updateTask(item TaskItem) *Task {
	t, ok := s.Tasks[item.ID()]
	if !ok {
		t = &Task{
			Schedule: TaskScheduleAnytime,
		}
	}
	t.ID = item.ID()

	if item.P.Title != nil {
		t.Title = *item.P.Title
	}
	if item.P.IsProject != nil {
		t.IsProject = bool(*item.P.IsProject)
	}
	if item.P.Status != nil {
		t.Status = *item.P.Status
	}
	if item.P.Index != nil {
		t.Index = *item.P.Index
	}
	if item.P.InTrash != nil {
		t.InTrash = *item.P.InTrash
	}
	if item.P.Schedule != nil {
		t.Schedule = *item.P.Schedule
	}
	if item.P.ScheduledDate != nil {
		t.ScheduledDate = item.P.ScheduledDate.Time()
	}
	if item.P.CompletionDate != nil {
		t.CompletionDate = item.P.CompletionDate.Time()
	}
	if item.P.DeadlineDate != nil {
		t.DeadlineDate = item.P.DeadlineDate.Time()
	}
	if item.P.CreationDate != nil {
		cd := item.P.CreationDate.Time()
		t.CreationDate = *cd
	}
	if item.P.ModificationDate != nil {
		t.ModificationDate = item.P.ModificationDate.Time()
	}
	if item.P.AreaIDs != nil {
		ids := *item.P.AreaIDs
		t.AreaIDs = ids
	}
	if item.P.ParentTaskIDs != nil {
		ids := *item.P.ParentTaskIDs
		t.ParentTaskIDs = ids
	}
	if item.P.Note != nil {
		t.Note = *item.P.Note
	}
	if item.P.Title != nil {
		t.Title = *item.P.Title
	}

	return t
}

// CheckListItemsByTask returns check lists associated with a particular item
func (s *State) CheckListItemsByTask(task *Task) []*CheckList {
	items := []*CheckList{}
	for _, item := range s.CheckListItems {
		if item.Status == TaskStatusCompleted {
			continue
		}
		isChild := false
		for _, taskID := range item.TaskIDs {
			isChild = isChild || task.ID == taskID
		}
		if isChild {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}

// CheckList describes a check list item
//0|uuid|TEXT|0||1
//1|userModificationDate|REAL|0||0
//2|creationDate|REAL|0||0
//3|title|TEXT|0||0
//4|status|INTEGER|0||0
//5|stopDate|REAL|0||0
//6|index|INTEGER|0||0
//7|task|TEXT|0||0
type CheckList struct {
	ID               string
	CreationDate     time.Time
	ModificationDate *time.Time
	Status           TaskStatus
	Title            string
	Index            int
	CompletionDate   *time.Time
	TaskIDs          []string
}

type CheckListItemPayload struct {
	CreationDate     *Timestamp  `json:"cd,omitempty"`
	ModificationDate *Timestamp  `json:"md,omitempty"`
	Index            *int        `json:"ix"`
	Status           *TaskStatus `json:"ss,omitempty"`
	Title            *string     `json:"tt,omitempty"`
	CompletionDate   *Timestamp  `json:"sp,omitempty"`
	TaskIDs          *[]string   `json:"ts,omitempty"`
}

// CheckListItem describes an event on a check list item
type CheckListItem struct {
	Item
	P CheckListItemPayload `json:"p"`
}

func (s *State) updateCheckListItem(item CheckListItem) *CheckList {
	c, ok := s.CheckListItems[item.ID]
	if !ok {
		c = &CheckList{}
	}
	c.ID = item.ID

	if item.P.CreationDate != nil {
		t := item.P.CreationDate.Time()
		c.CreationDate = *t
	}
	if item.P.ModificationDate != nil {
		c.ModificationDate = item.P.ModificationDate.Time()
	}
	if item.P.Index != nil {
		c.Index = *item.P.Index
	}
	if item.P.Title != nil {
		c.Title = *item.P.Title
	}
	if item.P.Status != nil {
		c.Status = *item.P.Status
	}
	if item.P.TaskIDs != nil {
		ids := *item.P.TaskIDs
		c.TaskIDs = ids
	}

	return c
}

// Area describes an Area inside things. An Area is a container for tasks
// 0|uuid|TEXT|0||1
// 1|title|TEXT|0||0
// 2|visible|INTEGER|0||0
// 3|index|INTEGER|0||0
type Area struct {
	ID    string
	Title string
	Tags  []*Tag
	Tasks []*Task
}

type AreaItemPayload struct {
	IX     *int     `json:"ix"`
	Title  *string  `json:"tt"`
	TagIDs []string `json:"tg"`
}

// AreaItem describes an event on an area
type AreaItem struct {
	Item
	P AreaItemPayload `json:"p"`
}

func (item AreaItem) ID() string {
	return item.Item.ID
}

func (s *State) updateArea(item AreaItem) *Area {
	a, ok := s.Areas[item.ID()]
	if !ok {
		a = &Area{}
	}
	a.ID = item.ID()

	if item.P.Title != nil {
		a.Title = *item.P.Title
	}

	return a
}

// Tag describes the aggregated state of an Tag
// 0|uuid|TEXT|0||1
// 1|title|TEXT|0||0
// 2|shortcut|TEXT|0||0
// 3|usedDate|REAL|0||0
// 4|parent|TEXT|0||0
// 5|index|INTEGER|0||0
type Tag struct {
	ID           string
	Title        string
	ParentTagIDs []string
	ShortHand    string
}

type TagItemPayload struct {
	IX           *int      `json:"ix"`
	Title        *string   `json:"tt"`
	ShortHand    *string   `json:"sh"`
	ParentTagIDs *[]string `json:"pn"`
}

// tagItem describes an event on a tag
type TagItem struct {
	Item
	P TagItemPayload `json:"p"`
}

func (t TagItem) ID() string {
	return t.Item.ID
}

// SubTags returns all child tags for a given root, ensuring sort order is kept intact
func (s *State) SubTags(root *Tag) []*Tag {
	children := []*Tag{}
	for _, tag := range s.Tags {
		if tag == root {
			continue
		}

		isChild := false
		for _, parentID := range tag.ParentTagIDs {
			isChild = isChild || parentID == root.ID
		}
		if isChild {
			children = append(children, tag)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].ShortHand < children[j].ShortHand
	})
	return children
}

func (s *State) updateTag(item TagItem) *Tag {
	t, ok := s.Tags[item.ID()]
	if !ok {
		t = &Tag{}
	}
	t.ID = item.ID()

	if item.P.Title != nil {
		t.Title = *item.P.Title
	}
	if item.P.ShortHand != nil {
		t.ShortHand = *item.P.ShortHand
	}
	if item.P.ParentTagIDs != nil {
		var ids []string = *item.P.ParentTagIDs
		t.ParentTagIDs = ids
	}

	return t
}

// Update applies all items to update the aggregated state
func (s *State) Update(items ...Item) error {
	for _, rawItem := range items {
		switch rawItem.Kind {
		case ItemKindTask:
			item := TaskItem{Item: rawItem}
			if err := json.Unmarshal(rawItem.P, &item.P); err != nil {
				return err
			}

			switch item.Action {
			case ItemActionCreated:
				fallthrough
			case ItemActionModified:
				s.Tasks[item.ID()] = s.updateTask(item)
			case ItemActionDeleted:
				delete(s.Tasks, item.ID())
			default:
				fmt.Printf("Action %q on %q is not implemented yet", item.Action, rawItem.Kind)
			}

		case ItemKindChecklistItem:
			item := CheckListItem{Item: rawItem}
			if err := json.Unmarshal(rawItem.P, &item.P); err != nil {
				return err
			}

			switch item.Action {
			case ItemActionCreated:
				fallthrough
			case ItemActionModified:
				s.CheckListItems[item.ID] = s.updateCheckListItem(item)
			case ItemActionDeleted:
				delete(s.CheckListItems, item.ID)
			default:
				fmt.Printf("Action %q on %q is not implemented yet", item.Action, rawItem.Kind)
			}

		case ItemKindArea:
			item := AreaItem{Item: rawItem}
			if err := json.Unmarshal(rawItem.P, &item.P); err != nil {
				return err
			}

			switch item.Action {
			case ItemActionCreated:
				fallthrough
			case ItemActionModified:
				s.Areas[item.ID()] = s.updateArea(item)

			case ItemActionDeleted:
				delete(s.Areas, item.ID())
			default:
				fmt.Printf("Action %q on %q is not implemented yet", item.Action, rawItem.Kind)
			}

		case ItemKindTag:
			item := TagItem{Item: rawItem}
			if err := json.Unmarshal(rawItem.P, &item.P); err != nil {
				return err
			}

			switch item.Action {
			case ItemActionCreated:
				fallthrough
			case ItemActionModified:
				s.Tags[item.ID()] = s.updateTag(item)
			case ItemActionDeleted:
				delete(s.Tags, item.ID())
			default:
				fmt.Printf("Action %q on %q is not implemented yet", item.Action, rawItem.Kind)
			}

		default:
			fmt.Printf("%q is not implemented yet\n", rawItem.Kind)
		}
	}
	return nil
}
