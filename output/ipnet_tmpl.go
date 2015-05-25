package output

var ipnetTmpl = `inetnum:       (IPNetwork)
aut-num:       {{.IPNetwork.Handle}}
abuse-c:       (handle)
owner:         {{.IPNetwork.Name}}
ownerid:       (CPF/CNPJ)
responsible:   {{.IPNetwork.Name}}
address:     
address:     
country:       {{.IPNetwork.Country}}
phone:       
start-address: {{.IPNetwork.StartAddress}}
end-address:   {{.IPNetwork.EndAddress}}
ip-version:    {{.IPNetwork.IPVersion}}
type:          {{.IPNetwork.Type}}
parent-handle: {{.IPNetwork.ParentHandle}}
status:        {{.IPNetwork.Status}}
owner-c:     
tech-c:      
inetrev:     
nserver:     
nsstat:      
nslastaa:    
created:     {{.CreatedAt}}
changed:     {{.UpdatedAt}}

{{range .ContactsInfos}}nic-hdl-br: {{.Handle}}
{{range .Persons}}person: {{.}}
{{end}}{{range .Emails}}e-mail: {{.}}
{{end}}{{range .Addresses}}address: {{.}}
{{end}}{{range .Phones}}phone: {{.}}
{{end}}created: {{.CreatedAt}}
changed: {{.UpdatedAt}}

{{end}}`