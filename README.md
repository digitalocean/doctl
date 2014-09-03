# GODO

Godo is a Go client library for accessing the DigitalOcean V2 API.

## Usage

```go
import "github.com/digitaloceancloud/godo"
```

Create a new DigitalOcean client, then use the exposed services to
access different parts of the DigitalOcean API.

### Authentication

Currently, Personal Access Token (PAT) is the only method of
authenticating with the API. You can manage your tokens
at the Digital Ocean Control Panel [Applications Page](https://cloud.digitalocean.com/settings/applications).

You can then use your token to creat a new client:

```go
import "code.google.com/p/goauth2/oauth"

pat := "mytoken"
t := &oauth.Transport{
	Token: &oauth.Token{AccessToken: pat},
}

client := godo.NewClient(t.Client())
```

## Examples

[Digital Ocean API Documentation](https://developers.digitalocean.com/v2/)


To list all Droplets your account has access to:

```go
droplets, _, err := client.Droplet.List()
if err != nil {
	fmt.Printf("error: %v\n\n", err)
	return err
} else {
	fmt.Printf("%v\n\n", godo.Stringify(droplets))
}
```

To create a new Droplet:

```go
dropletName := "super-cool-droplet"

createRequest := &godo.DropletCreateRequest{
	Name:   godo.String(dropletName),
	Region: godo.String("nyc2"),
	Size:   godo.String("512mb"),
	Image:  godo.Int(3240036), // ubuntu 14.04 64bit
}

newDroplet, _, err := client.Droplet.Create(createRequest)

if err != nil {
	fmt.Printf("Something bad happened: %s\n\n", err)
	return err
}
```
