//go:build darwin

package common

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework UniformTypeIdentifiers

#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>

typedef struct {
	const char *path;
	const char *error;
} FilePickerResult;

// openDatabaseFilePanel shows NSOpenPanel for a database file and activates
// the security-scoped resource so the sandbox grants access to sibling files
// (journal, WAL, SHM) via Related Items.
static FilePickerResult openDatabaseFilePanel(const char *title, const char *filters) {
	__block FilePickerResult result = {NULL, NULL};

	dispatch_sync(dispatch_get_main_queue(), ^{
		NSOpenPanel *panel = [NSOpenPanel openPanel];
		[panel setTitle:[NSString stringWithUTF8String:title]];
		[panel setCanChooseFiles:YES];
		[panel setCanChooseDirectories:NO];
		[panel setAllowsMultipleSelection:NO];

		// Parse semicolon-separated extensions
		NSString *filterStr = [NSString stringWithUTF8String:filters];
		NSArray *extensions = [filterStr componentsSeparatedByString:@";"];
		NSMutableArray *contentTypes = [NSMutableArray array];
		for (NSString *ext in extensions) {
			NSString *trimmed = [ext stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
			if ([trimmed length] > 0) {
				UTType *t = [UTType typeWithFilenameExtension:trimmed];
				if (t != nil) {
					[contentTypes addObject:t];
				}
			}
		}
		if ([contentTypes count] > 0) {
			[panel setAllowedContentTypes:contentTypes];
		}

		NSModalResponse response = [panel runModal];
		if (response != NSModalResponseOK) {
			return;
		}

		NSURL *url = [[panel URLs] firstObject];
		if (url == nil) {
			return;
		}

		// Activate security-scoped resource — this is what grants access to
		// Related Items (sibling -journal/-wal/-shm files) in the sandbox.
		[url startAccessingSecurityScopedResource];

		// Store a security-scoped bookmark so we can re-access on next launch.
		NSError *bookmarkError = nil;
		NSData *bookmark = [url bookmarkDataWithOptions:NSURLBookmarkCreationWithSecurityScope
						includingResourceValuesForKeys:nil
										 relativeToURL:nil
												 error:&bookmarkError];
		if (bookmark != nil) {
			NSString *key = [NSString stringWithFormat:@"bookmark_%@", [url path]];
			[[NSUserDefaults standardUserDefaults] setObject:bookmark forKey:key];
		}

		result.path = strdup([[url path] UTF8String]);
	});

	return result;
}

// resolveBookmark resolves a stored security-scoped bookmark for a path.
// Returns the resolved path if the bookmark is valid, NULL otherwise.
static const char* resolveBookmark(const char *originalPath) {
	__block const char *resolved = NULL;

	dispatch_sync(dispatch_get_main_queue(), ^{
		NSString *key = [NSString stringWithFormat:@"bookmark_%s", originalPath];
		NSData *bookmark = [[NSUserDefaults standardUserDefaults] objectForKey:key];
		if (bookmark == nil) {
			return;
		}

		BOOL isStale = NO;
		NSError *error = nil;
		NSURL *url = [NSURL URLByResolvingBookmarkData:bookmark
											   options:NSURLBookmarkResolutionWithSecurityScope
										 relativeToURL:nil
								   bookmarkDataIsStale:&isStale
												 error:&error];
		if (url == nil || error != nil) {
			return;
		}

		if (isStale) {
			// Re-create the bookmark
			NSData *newBookmark = [url bookmarkDataWithOptions:NSURLBookmarkCreationWithSecurityScope
								includingResourceValuesForKeys:nil
												 relativeToURL:nil
														 error:nil];
			if (newBookmark != nil) {
				[[NSUserDefaults standardUserDefaults] setObject:newBookmark forKey:key];
			}
		}

		[url startAccessingSecurityScopedResource];
		resolved = strdup([[url path] UTF8String]);
	});

	return resolved;
}

// stopAccessingPath stops accessing a security-scoped resource for a path.
static void stopAccessingPath(const char *path) {
	NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
	[url stopAccessingSecurityScopedResource];
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

// selectDatabaseFileDarwin uses NSOpenPanel with security-scoped resource
// activation so the sandbox grants access to SQLite auxiliary files.
func (a *App) selectDatabaseFileDarwin(dbType string) (string, error) {
	cfg, ok := databaseFileConfigs[dbType]
	if !ok {
		return "", fmt.Errorf("unsupported file-based database type: %s", dbType)
	}

	exts := buildExtensionList(cfg)
	cTitle := C.CString(cfg.Title)
	cFilters := C.CString(exts)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cFilters))

	result := C.openDatabaseFilePanel(cTitle, cFilters)

	if result.error != nil {
		errStr := C.GoString(result.error)
		C.free(unsafe.Pointer(result.error))
		return "", fmt.Errorf("%s", errStr)
	}

	if result.path == nil {
		return "", nil
	}

	path := C.GoString(result.path)
	C.free(unsafe.Pointer(result.path))

	ext := strings.ToLower(path)
	dotIndex := strings.LastIndex(ext, ".")
	if dotIndex == -1 || dotIndex == len(ext)-1 {
		return "", fmt.Errorf("invalid file type for %s", dbType)
	}
	ext = ext[dotIndex+1:]

	if !cfg.Extensions[ext] {
		return "", fmt.Errorf("invalid file type for %s", dbType)
	}

	return path, nil
}

// ResolveDatabaseBookmark re-activates a security-scoped bookmark for a
// previously opened database path. Call this on app startup for auto-reconnect.
func (a *App) ResolveDatabaseBookmark(path string) (string, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	resolved := C.resolveBookmark(cPath)
	if resolved == nil {
		return "", fmt.Errorf("no valid bookmark for path: %s", path)
	}

	resolvedPath := C.GoString(resolved)
	C.free(unsafe.Pointer(resolved))
	return resolvedPath, nil
}

// StopAccessingDatabase stops accessing a security-scoped resource.
func (a *App) StopAccessingDatabase(path string) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	C.stopAccessingPath(cPath)
}

func buildExtensionList(cfg databaseFileConfig) string {
	parts := strings.Split(cfg.Pattern, ";")
	var exts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "*.")
		if p != "" {
			exts = append(exts, p)
		}
	}
	return strings.Join(exts, ";")
}
