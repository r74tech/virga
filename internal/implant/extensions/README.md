# Extension System Architecture

## Extension Priority (Production Use)

### Recommended for Production
1. **BOF (Beacon Object Files)** - Primary extension type
   - No dependencies
   - In-memory execution
   - Stealthy operation
   - Examples: Process injection, token manipulation

2. **Native Extensions** - Secondary option
   - Compiled binaries (.dll/.so)
   - No interpreter needed
   - Platform-specific
   - Examples: Keyloggers, screenshot tools

### Limited Use Cases
3. **Alias Extensions**
   - Simple command wrappers
   - Relies on system shell
   - Use for convenience only

### Development/Testing Only
4. **Script Extensions**
   - **NOT RECOMMENDED FOR PRODUCTION**
   - Requires interpreter on target
   - Easily detected by EDR
   - Use only in controlled environments

## Why Keep Script Extensions?

1. **Development Speed** - Quick prototyping before converting to BOF
2. **Testing** - Validate logic before compilation
3. **Special Environments** - Known systems with interpreters
4. **Educational** - Learning the extension system

## Production Recommendations

For real operations, always use:
```
BOF > Native > Alias >> Script (avoid)
```
