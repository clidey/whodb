# WhoDB - Super Simple Setup

## 🎯 Quick Commands

### Run Production Mode
```bash
# Community Edition (PostgreSQL, MySQL, MongoDB, etc.)
./run.sh

# Enterprise Edition (adds Oracle, MSSQL, DynamoDB)
./run.sh --ee
```

### Run Development Mode (with hot-reload)
```bash
# Community Edition
./dev.sh

# Enterprise Edition
./dev.sh --ee
```

### Build Binaries
```bash
# Community Edition
./build.sh

# Enterprise Edition
./build.sh --ee
```

### Frontend Only (for development)
```bash
# Community Edition
cd frontend && pnpm run start

# Enterprise Edition
cd frontend && pnpm run start:ee
```

## 📝 That's it!

- Production mode: http://localhost:8080
- Development mode: Backend on :8080, Frontend on :1234 with hot-reload
- No complex setup needed
- Switch between editions with just a flag

## 🔧 Manual Commands (if scripts don't work)

**Run CE:**
```bash
cd core && go run .
```

**Run EE:**
```bash
cd core && go run -tags ee .
```

**Build CE:**
```bash
cd core && go build -o whodb
```

**Build EE:**
```bash
cd core && go build -tags ee -o whodb-ee
```