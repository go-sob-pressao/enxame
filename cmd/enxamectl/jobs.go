package main

import "net/url"

func jobInsert(c *cliente, args []string) error {
	fs := novoFlagSet("job insert")
	fila := fs.String("queue", "default", "fila")
	kind := fs.String("kind", "", "kind do job")
	a := fs.String("args", "", "argumentos: JSON ou @arquivo")
	chave := fs.String("unique-key", "", "chave única")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "kind"); err != nil {
		return err
	}
	v, err := jsonDe(*a)
	if err != nil {
		return err
	}
	return c.chamar("POST", "/v1/jobs", map[string]any{
		"queue": *fila, "kind": *kind, "args": v,
		"unique_key": *chave})
}

func jobDescribe(c *cliente, args []string) error {
	fs := novoFlagSet("job describe")
	jid := fs.String("id", "", "id do job")
	espera := fs.Duration("wait", 0, "espera o estado final")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "id"); err != nil {
		return err
	}
	caminho := "/v1/jobs/" + url.PathEscape(*jid)
	if *espera > 0 {
		caminho += "?wait=" + espera.String()
	}
	return c.chamar("GET", caminho, nil)
}

func jobCancel(c *cliente, args []string) error {
	fs := novoFlagSet("job cancel")
	jid := fs.String("id", "", "id do job")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "id"); err != nil {
		return err
	}
	return c.chamar("POST", "/v1/jobs/"+url.PathEscape(*jid)+"/cancel",
		nil)
}

func workflowStart(c *cliente, args []string) error {
	fs := novoFlagSet("workflow start")
	tipo := fs.String("type", "", "tipo do workflow")
	wid := fs.String("id", "", "workflow_id, escolhido pela aplicação")
	in := fs.String("input", "", "entrada: JSON ou @arquivo")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "type", "id"); err != nil {
		return err
	}
	v, err := jsonDe(*in)
	if err != nil {
		return err
	}
	return c.chamar("POST", "/v1/workflows", map[string]any{
		"type": *tipo, "workflow_id": *wid, "input": v})
}

func workflowDescribe(c *cliente, args []string) error {
	fs := novoFlagSet("workflow describe")
	run := fs.String("run", "", "id do run")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := obrigatorio(fs, "run"); err != nil {
		return err
	}
	return c.chamar("GET", "/v1/workflows/"+url.PathEscape(*run), nil)
}
