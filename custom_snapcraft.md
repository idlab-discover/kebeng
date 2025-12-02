# Custom snapcraft
## 1. Clone the snapcraft repo
```bash
git clone https://github.com/canonical/snapcraft.git
cd snapcraft
```

## 2. Modify the store url
`snapcraft` points default to Canonical's Snap Store.
We want to modify snapcraft in such a way it points to our self-hosted store.
We can do this by modifying the `/store/constants.py` file with the following code:
```python
import os

ip = os.getenv("STORE_IP")
if not ip:
    print("Warning: STORE_IP not set, using default value")
    ip = "dashboard.snapcraft.io"

STORE_URL: Final[str] = "http://"+ip+":8080"
"""Default store backend URL."""

STORE_UPLOAD_URL: Final[str] = "http://"+ip+":8080"
"""Default store upload URL."""
```
![store_url](/img/snapcraft_store_url.png)

## 3. Create package
Use the `snapcraft` command in the root of the snapd repository to create a snap package of the modified snapd code.
This modified snapd package can later be installed on devices that will use your Kebeng store.