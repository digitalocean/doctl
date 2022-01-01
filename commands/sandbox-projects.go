package commands

import (
    "github.com/digitalocean/doctl"
    "github.com/spf13/cobra"
)

func SandboxProjects() *Command {
    cmd := &Command{
        Command: &cobra.Command{
            Use: "sandbox-projects",
            Short: "This is the sandbox-projects subtree",
            Long: `This is more information about the sandbox-projects subtree`,
        },
    }

    Create := cmdBuilderWithInit(cmd, RunSandboxProjectsCreate, "create [<project>]", "Create a Nimbella Project",
        `More information about 'sandbox-projects create'`,
        Writer, false)
    AddBoolFlag(Create, "config", "", false, "Generate template config file")
    AddStringFlag(Create, "type", "", "", "API specs source")
    AddStringFlag(Create, "language", "l", "js", "Language for the project (creates sample project unless source is specified)")
    AddBoolFlag(Create, "overwrite", "", false, "Overwrites the existing file(s)")
    AddStringFlag(Create, "debug", "", "", "Debug level output")
    Create.Flags().MarkHidden("debug")

    Deploy := cmdBuilderWithInit(cmd, RunSandboxProjectsDeploy, "deploy [<projects>]", "Deploy Nimbella projects",
        `More information about 'sandbox-projects deploy'`,
        Writer, false)
    AddStringFlag(Deploy, "target", "", "", "The target namespace")
    AddStringFlag(Deploy, "env", "", "", "Path to runtime environment file")
    AddStringFlag(Deploy, "build-env", "", "", "Path to build-time environment file")
    AddStringFlag(Deploy, "apihost", "", "", "API host to use")
    AddStringFlag(Deploy, "auth", "", "", "OpenWhisk auth token to use")
    AddBoolFlag(Deploy, "insecure", "", false, "Ignore SSL Certificates")
    AddBoolFlag(Deploy, "verbose-build", "", false, "Display build details")
    AddBoolFlag(Deploy, "verbose-zip", "", false, "Display start/end of zipping phase for each action")
    AddBoolFlag(Deploy, "production", "", false, "Deploy to the production namespace instead of the test one")
    AddBoolFlag(Deploy, "yarn", "", false, "Use yarn instead of npm for node builds")
    AddStringFlag(Deploy, "web-local", "", "", "A local directory to receive web deploy, instead of uploading")
    AddStringFlag(Deploy, "include", "", "", "Project portions to include")
    AddStringFlag(Deploy, "exclude", "", "", "Project portions to exclude")
    AddBoolFlag(Deploy, "remote-build", "", false, "Run builds remotely")
    AddBoolFlag(Deploy, "incremental", "", false, "Deploy only changes since last deploy")
    AddBoolFlag(Deploy, "anon-github", "", false, "Attempt GitHub deploys anonymously")
    AddStringFlag(Deploy, "debug", "", "", "Debug level output")
    Deploy.Flags().MarkHidden("debug")

    GetMetadata := cmdBuilderWithInit(cmd, RunSandboxProjectsGetMetadata, "get-metadata [<project>]", "Obtain metadata of a Nimbella project",
        `More information about 'sandbox-projects get-metadata'`,
        Writer, false)
    AddStringFlag(GetMetadata, "env", "", "", "Path to environment file")
    AddStringFlag(GetMetadata, "include", "", "", "Project portions to include")
    AddStringFlag(GetMetadata, "exclude", "", "", "Project portions to exclude")
    AddStringFlag(GetMetadata, "debug", "", "", "Debug level output")
    GetMetadata.Flags().MarkHidden("debug")

    ServeWeb := cmdBuilderWithInit(cmd, RunSandboxProjectsServeWeb, "serve-web <location>", "Serves content from the local Web folder, proxying API requests to given/current namespace",
        `More information about 'sandbox-projects serve-web'`,
        Writer, false)
    AddStringFlag(ServeWeb, "namespace", "", "", "The namespace to proxy (current namespace if omitted)")
    AddStringFlag(ServeWeb, "apihost", "", "", "API host of the namespace")
    AddIntFlag(ServeWeb, "port", "", 8080, "The port of the web server (default is 8080)")
    AddStringFlag(ServeWeb, "debug", "", "", "Debug level output")
    ServeWeb.Flags().MarkHidden("debug")

    Update := cmdBuilderWithInit(cmd, RunSandboxProjectsUpdate, "update [<project>]", "Update a Nimbella Project",
        `More information about 'sandbox-projects update'`,
        Writer, false)
    AddBoolFlag(Update, "config", "", false, "Generate config file")
    AddStringFlag(Update, "type", "", "", "API specs source")
    AddStringFlag(Update, "language", "l", "js", "Language for the project (creates sample project unless source is specified)")
    AddBoolFlag(Update, "overwrite", "", false, "Overwrites the existing file(s)")
    AddStringFlag(Update, "debug", "", "", "Debug level output")
    Update.Flags().MarkHidden("debug")

    Watch := cmdBuilderWithInit(cmd, RunSandboxProjectsWatch, "watch [<projects>]", "Watch Nimbella projects, deploying incrementally on change",
        `More information about 'sandbox-projects watch'`,
        Writer, false)
    AddStringFlag(Watch, "target", "", "", "The target namespace")
    AddStringFlag(Watch, "env", "", "", "Path to runtime environment file")
    AddStringFlag(Watch, "build-env", "", "", "Path to build-time environment file")
    AddStringFlag(Watch, "apihost", "", "", "API host to use")
    AddStringFlag(Watch, "auth", "", "", "OpenWhisk auth token to use")
    AddBoolFlag(Watch, "insecure", "", false, "Ignore SSL Certificates")
    AddBoolFlag(Watch, "verbose-build", "", false, "Display build details")
    AddBoolFlag(Watch, "verbose-zip", "", false, "Display start/end of zipping phase for each action")
    AddBoolFlag(Watch, "yarn", "", false, "Use yarn instead of npm for node builds")
    AddStringFlag(Watch, "web-local", "", "", "A local directory to receive web deploy, instead of uploading")
    AddStringFlag(Watch, "include", "", "", "Project portions to include")
    AddStringFlag(Watch, "exclude", "", "", "Project portions to exclude")
    AddBoolFlag(Watch, "remote-build", "", false, "Run builds remotely")
    AddStringFlag(Watch, "debug", "", "", "Debug level output")
    Watch.Flags().MarkHidden("debug")

    return cmd
}
func RunSandboxProjectsCreate(c *CmdConfig) error {
    argCount := len(c.Args)
    if argCount > 1 {
        return doctl.NewTooManyArgsErr(c.NS)
    }
    output, err := RunSandboxExec("project/create", c, []string{"config","overwrite"}, []string{"type","language","debug"})
    if err != nil {
        return err
    }
    PrintSandboxTextOutput(output)
    return nil
}

func RunSandboxProjectsDeploy(c *CmdConfig) error {
    output, err := RunSandboxExec("project/deploy", c, []string{"insecure","verbose-build","verbose-zip","production","yarn","remote-build","incremental","anon-github"}, []string{"target","env","build-env","apihost","auth","web-local","include","exclude","debug"})
    if err != nil {
        return err
    }
    PrintSandboxTextOutput(output)
    return nil
}

func RunSandboxProjectsGetMetadata(c *CmdConfig) error {
    argCount := len(c.Args)
    if argCount > 1 {
        return doctl.NewTooManyArgsErr(c.NS)
    }
    output, err := RunSandboxExec("project/get-metadata", c, []string{}, []string{"env","include","exclude","debug"})
    if err != nil {
        return err
    }
    PrintSandboxTextOutput(output)
    return nil
}

func RunSandboxProjectsServeWeb(c *CmdConfig) error {
    err := ensureOneArg(c)
    if err != nil {
        return err
    }
    output, err := RunSandboxExec("project/serve-web", c, []string{}, []string{"namespace","apihost","port","debug"})
    if err != nil {
        return err
    }
    PrintSandboxTextOutput(output)
    return nil
}

func RunSandboxProjectsUpdate(c *CmdConfig) error {
    argCount := len(c.Args)
    if argCount > 1 {
        return doctl.NewTooManyArgsErr(c.NS)
    }
    output, err := RunSandboxExec("project/update", c, []string{"config","overwrite"}, []string{"type","language","debug"})
    if err != nil {
        return err
    }
    PrintSandboxTextOutput(output)
    return nil
}

func RunSandboxProjectsWatch(c *CmdConfig) error {
    argCount := len(c.Args)
    if argCount > 1 {
        return doctl.NewTooManyArgsErr(c.NS)
    }
    output, err := RunSandboxExec("project/watch", c, []string{"insecure","verbose-build","verbose-zip","yarn","remote-build"}, []string{"target","env","build-env","apihost","auth","web-local","include","exclude","debug"})
    if err != nil {
        return err
    }
    PrintSandboxTextOutput(output)
    return nil
}
