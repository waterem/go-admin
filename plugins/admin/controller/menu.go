package controller

import (
	"bytes"
	"encoding/json"
	"github.com/chenhg5/go-admin/context"
	"github.com/chenhg5/go-admin/modules/auth"
	"github.com/chenhg5/go-admin/modules/connections"
	"github.com/chenhg5/go-admin/modules/menu"
	"github.com/chenhg5/go-admin/plugins/admin/models"
	"github.com/chenhg5/go-admin/template"
	"github.com/chenhg5/go-admin/template/types"
	template2 "html/template"
	"net/http"
)

// 显示菜单
func ShowMenu(ctx *context.Context) {
	defer GlobalDeferHandler(ctx)
	GetMenuInfoPanel(ctx)
	return
}

// 显示编辑菜单
func ShowEditMenu(ctx *context.Context) {
	id := ctx.Request.URL.Query().Get("id")
	formData, title, description := models.TableList["menu"].GetDataFromDatabaseWithId("menu", id)

	tmpl, tmplName := template.Get("adminlte").GetTemplate(ctx.Request.Header.Get("X-PJAX") == "true")

	path := ctx.Path()
	menu.GlobalMenu.SetActiveClass(path)

	ctx.Response.Header.Add("Content-Type", "text/html; charset=utf-8")
	user := ctx.UserValue["user"].(auth.User)

	js := `<script>
$('.icon').iconpicker({placement: 'bottomLeft'});
</script>`

	buf := new(bytes.Buffer)
	tmpl.ExecuteTemplate(buf, tmplName, types.Page{
		User: user,
		Menu: menu.GetGlobalMenu(user),
		System: types.SystemInfo{
			"0.0.1",
		},
		Panel: types.Panel{
			Content: template.Get(Config.THEME).Form().
				SetContent(formData).
				SetPrefix(Config.PREFIX).
				SetUrl(Config.PREFIX+"/menu/edit").
				SetToken(auth.TokenHelper.AddToken()).
				SetInfoUrl(Config.PREFIX+"/menu").
				GetContent() + template2.HTML(js),
			Description: description,
			Title:       title,
		},
		AssertRootUrl: Config.PREFIX,
		Title:         Config.TITLE,
		Logo:          Config.LOGO,
		MiniLogo:      Config.MINILOGO,
	})
	ctx.WriteString(buf.String())
}

// 删除菜单
func DeleteMenu(ctx *context.Context) {
	id := ctx.Request.URL.Query().Get("id")
	user := ctx.UserValue["user"].(auth.User)

	buffer := new(bytes.Buffer)

	connections.GetConnection().Exec("delete from goadmin_menu where id = ?", id)

	menu.SetGlobalMenu(user)

	ctx.WriteString(buffer.String())

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
}

// 编辑菜单
func EditMenu(ctx *context.Context) {
	defer GlobalDeferHandler(ctx)

	id := ctx.Request.FormValue("id")
	title := ctx.Request.FormValue("title")
	parentId := ctx.Request.FormValue("parent_id")
	if parentId == "" {
		parentId = "0"
	}
	icon := ctx.Request.FormValue("icon")
	uri := ctx.Request.FormValue("uri")

	roles := ctx.Request.Form["roles[]"]

	for _, roleId := range roles {
		checkRoleMenu, _ := connections.GetConnection().Query("select * from goadmin_role_menu where role_id = ? and menu_id = ?", roleId, id)
		if len(checkRoleMenu) < 1 {
			connections.GetConnection().Exec("insert into goadmin_role_menu (menu_id, role_id) values (?, ?)", id, roleId)
		}
	}

	connections.GetConnection().Exec("update goadmin_menu set title = ?, parent_id = ?, icon = ?, uri = ? where id = ?",
		title, parentId, icon, uri, id)

	menu.SetGlobalMenu(ctx.UserValue["user"].(auth.User))

	GetMenuInfoPanel(ctx)
	ctx.Response.Header.Add("Content-Type", "text/html; charset=utf-8")
	ctx.Response.Header.Add("X-PJAX-URL", Config.PREFIX+"/menu")
}

// 新建菜单
func NewMenu(ctx *context.Context) {
	defer GlobalDeferHandler(ctx)

	title := ctx.Request.FormValue("title")
	parentId := ctx.Request.FormValue("parent_id")
	if parentId == "" {
		parentId = "0"
	}
	icon := ctx.Request.FormValue("icon")
	uri := ctx.Request.FormValue("uri")

	user := ctx.UserValue["user"].(auth.User)

	res := connections.GetConnection().Exec("insert into goadmin_menu (title, parent_id, icon, uri, `order`) values (?, ?, ?, ?, ?)",
		title, parentId, icon, uri, (menu.GetGlobalMenu(user)).MaxOrder+1)

	roles := ctx.Request.Form["roles[]"]

	id, _ := res.LastInsertId()

	for _, roleId := range roles {
		connections.GetConnection().Exec("insert into goadmin_role_menu (menu_id, role_id) values (?, ?)", id, roleId)
	}

	globalMenu := menu.GetGlobalMenu(user)
	(globalMenu).SexMaxOrder(globalMenu.MaxOrder + 1)
	menu.SetGlobalMenu(user)

	GetMenuInfoPanel(ctx)
	ctx.Response.Header.Add("Content-Type", "text/html; charset=utf-8")
	ctx.Response.Header.Add("X-PJAX-URL", Config.PREFIX+"/menu")
}

// 修改菜单顺序
func MenuOrder(ctx *context.Context) {
	defer GlobalDeferHandler(ctx)

	var data []map[string]interface{}
	json.Unmarshal([]byte(ctx.Request.FormValue("_order")), &data)

	count := 1
	for _, v := range data {
		if child, ok := v["children"]; ok {
			connections.GetConnection().Exec("update goadmin_menu set `order` = ? where id = ?", count, v["id"])
			for _, v2 := range child.([]interface{}) {
				connections.GetConnection().Exec("update goadmin_menu set `order` = ? where id = ?", count, v2.(map[string]interface{})["id"])
				count++
			}
		} else {
			connections.GetConnection().Exec("update goadmin_menu set `order` = ? where id = ?", count, v["id"])
			count++
		}
	}
	menu.SetGlobalMenu(ctx.UserValue["user"].(auth.User))

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
	return
}

func GetMenuInfoPanel(ctx *context.Context) {
	path := ctx.Path()
	user := ctx.UserValue["user"].(auth.User)

	menu.GlobalMenu.SetActiveClass(path)

	editUrl := Config.PREFIX + "/menu/edit/show"
	deleteUrl := Config.PREFIX + "/menu/delete"
	orderUrl := Config.PREFIX + "/menu/order"

	tree := template.Get(Config.THEME).Tree().SetTree((menu.GetGlobalMenu(user)).GlobalMenuList).
		SetEditUrl(editUrl).SetDeleteUrl(deleteUrl).SetOrderUrl(orderUrl).GetContent()
	header := template.Get(Config.THEME).Tree().GetTreeHeader()
	box := template.Get(Config.THEME).Box().SetHeader(header).SetBody(tree).GetContent()
	col1 := template.Get(Config.THEME).Col().SetSize(map[string]string{"md": "6"}).SetContent(box).GetContent()

	newForm := template.Get(Config.THEME).Form().SetPrefix(Config.PREFIX).SetUrl(Config.PREFIX + "/menu/new").
		SetInfoUrl(Config.PREFIX + "/menu").SetTitle("New").
		SetContent(models.GetNewFormList(models.TableList["menu"].Form.FormList)).GetContent()
	col2 := template.Get(Config.THEME).Col().SetSize(map[string]string{"md": "6"}).SetContent(newForm).GetContent()

	row := template.Get(Config.THEME).Row().SetContent(col1 + col2).GetContent()

	tmpl, tmplName := template.Get("adminlte").GetTemplate(ctx.Request.Header.Get("X-PJAX") == "true")

	menu.GlobalMenu.SetActiveClass(path)

	ctx.Response.Header.Add("Content-Type", "text/html; charset=utf-8")

	buf := new(bytes.Buffer)

	tmpl.ExecuteTemplate(buf, tmplName, types.Page{
		User: user,
		Menu: menu.GetGlobalMenu(user),
		System: types.SystemInfo{
			"0.0.1",
		},
		Panel: types.Panel{
			Content:     row,
			Description: "Menus Manage",
			Title:       "Menus Manage",
		},
		AssertRootUrl: Config.PREFIX,
		Title:         Config.TITLE,
		Logo:          Config.LOGO,
		MiniLogo:      Config.MINILOGO,
	})

	ctx.WriteString(buf.String())
}
