package model

type SysMenu struct {
	ID         int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentID   int64      `gorm:"column:parent_id;default:0" json:"parentId"`
	MenuKey    string     `gorm:"column:menu_key;size:64;uniqueIndex" json:"menuKey"`
	Title      string     `gorm:"column:title;size:100" json:"title"`
	Path       string     `gorm:"column:path;size:255" json:"path"`
	Icon       string     `gorm:"column:icon;size:64" json:"icon"`
	Permission string     `gorm:"column:permission;size:100" json:"permission"`
	MenuType   string     `gorm:"column:menu_type;size:20;default:MENU" json:"menuType"` // DIRECTORY, MENU, IFRAME, LINK
	Sort       int        `gorm:"column:sort;default:0" json:"sort"`
	IsHidden   int        `gorm:"column:is_hidden;default:0" json:"isHidden"` // 0: 否, 1: 是
	Status     string     `gorm:"column:status;size:20;default:ENABLE" json:"status"` // ENABLE, DISABLE
	CreateTime int64      `gorm:"column:create_time" json:"createTime"`
	UpdateTime int64      `gorm:"column:update_time" json:"updateTime"`
	Children   []*SysMenu `gorm:"-" json:"children,omitempty"`
}

func (SysMenu) TableName() string {
	return "sys_menu"
}
