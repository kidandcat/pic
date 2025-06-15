# Screenshot Implementation Plan

## Goal
Add 3 example screenshots to both README.md and docs/index.html landing page, displayed in a row with clickable full-screen capability.

## Website Selection
1. **example.com** - Simple, clean website perfect for demonstrating basic capture
2. **wikipedia.org** - Content-rich site showing text and images
3. **github.com** - Modern web app with complex UI elements

## Implementation Steps

### 1. Screenshot Capture
- Use browser automation to navigate to each site
- Capture full-page screenshots (tall format)
- Save screenshots with descriptive names in a dedicated directory

### 2. Directory Structure
- Create `docs/screenshots/` directory for storing images
- Name convention: `pic-example-{sitename}.png`

### 3. README.md Update
- Add a new "Examples" section
- Display 3 screenshots in a horizontal row
- Use markdown image syntax with links for full-screen viewing
- Keep images small (33% width each) for initial display

### 4. docs/index.html Update
- Add an "Example Captures" section
- Use responsive CSS grid/flexbox for 3-column layout
- Implement click-to-fullscreen functionality
- Ensure mobile responsiveness

### 5. Image Optimization
- Screenshots should be high quality but optimized for web
- Consider using CSS for responsive sizing rather than resizing images

## Technical Considerations
- Use relative paths for GitHub compatibility
- Ensure screenshots work in both light and dark modes
- Add alt text for accessibility