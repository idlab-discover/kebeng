# Custom snapd
## 1. Clone the snapd repo
```bash
https://github.com/canonical/snapd.git
cd snapd
```

## 2. Modify the store url
`snapd` points default to Canonical's Snap Store.
We want to modify snapd in such a way it points to our self-hosted store.
You can do this by making a total of 3 changes in 2 different files.

### Changes in /store/store.go
1. Replace the `apiURL()` function with the following code
```go
// apiURL returns the system default base API URL.
func apiURL() *url.URL {
	ip := os.Getenv("STORE_IP")
	s := fmt.Sprintf("http://%s:8080/", ip)
	if snapdenv.UseStagingStore() {
		s = fmt.Sprintf("http://%s:8080/", ip)
	}
	u, _ := url.Parse(s)
	return u
}
```
![apiurl](/img/snapd_store_apiurl.png)

2. Replace the `storeDeveloperURL()` function with the following code:
```go
var defaultStoreDeveloperURL = os.Getenv("STORE_IP")

func storeDeveloperURL() string {
	if snapdenv.UseStagingStore() {
		return defaultStoreDeveloperURL
	}
	return defaultStoreDeveloperURL
}
```
![developerurl](/img/snapd_store_developerurl.png)

### Changes in /overlord/devicestate/handlers_serial.go
Replace the `baseURL()` function with the following code:
```go
func baseURL() *url.URL {
	ip := os.Getenv("STORE_IP")
	fmt.Printf("STORE_IP: %q\n", ip)
	if snapdenv.UseStagingStore() {
		return mustParse(fmt.Sprintf("http://%s:8080/", ip))
	}
	return mustParse(fmt.Sprintf("https://%s/", ip))
}
```
![baseurl](/img/snapd_overlord_baseurl.png)


## 3. Add the root assertions
1. Run the Kebeng store with the following command:
```bash
docker compose -f docker-compose.yml down -v --remove-orphans
docker compose -f docker-compose.yml up --build
```

2. Login to the assertion database with the credentials for the database set in the config of the account service.

3. 