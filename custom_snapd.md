# Custom snapd
## 1. Clone the snapd repo
```bash
git clone https://github.com/canonical/snapd.git
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
var defaultStoreDeveloperURL = fmt.Sprintf("http://%s:8080/", os.Getenv("STORE_IP"))

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
	return mustParse(fmt.Sprintf("http://%s:8080/", ip))
}
```
![baseurl](/img/snapd_overlord_baseurl.png)


## 3. Add the root assertions
1. Run the Kebeng store with the following command:
```bash
docker compose -f docker-compose.yml down -v --remove-orphans
docker compose -f docker-compose.yml up --build
```

2. Login to the assertion database with the credentials for the database set in the config of the assertion service.

3. Retrieve the 2 automatically created assertions in the `account` and `accountkey` collections.

4. Add the assertions to the `const` in `/asserts/sysdb/trusted.go`. It should look like this:
![trusted_assertions](/img/trusted_assertions.png)

5. Replace the `init()` function in `/asserts/sysdb/trusted.go` with the following code:
```go
func init() {
    canonicalAccount, err := asserts.Decode([]byte(encodedCanonicalAccount))
    if err != nil {
        panic(fmt.Sprintf("cannot decode trusted assertion: %v", err))
    }
    canonicalRootAccountKey, err := asserts.Decode([]byte(encodedCanonicalRootAccountKey))
    if err != nil {
        panic(fmt.Sprintf("cannot decode trusted assertion: %v", err))
    }
    kebengRootAccount, err := asserts.Decode([]byte(encodedKebengRootAccount))
    if err != nil {
        panic(fmt.Sprintf("cannot decode trusted assertion: %v", err))
    }
    kebengRootAccountKey, err := asserts.Decode([]byte(encodedKebengRootAccountKey))
    if err != nil {
        panic(fmt.Sprintf("cannot decode trusted assertion: %v", err))
    }

    trustedAssertions = []asserts.Assertion{canonicalAccount, canonicalRootAccountKey, kebengRootAccount, kebengRootAccountKey}
}
```
Once the assertions are added, snapd trusts your Kebeng store as root.

## 4. Create package
Use the `snapcraft` command in the root of the snapd repository to create a snap package of the modified snapd code.
This modified snapd package can later be installed on devices that will use your Kebeng store. 