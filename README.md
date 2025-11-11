# 📚 CBZ Converter

![Go version](https://img.shields.io/github/go-mod/go-version/Romaixn/cbz-converter)
![License](https://img.shields.io/github/license/Romaixn/cbz-converter)
![GitHub Release](https://img.shields.io/github/v/release/Romaixn/cbz-converter)

## ✨ Features

- 🔄 Converts CBR files to CBZ format
- 📄 Converts PDF files to CBZ format
- 🔢 Renames image files with leading zeros for proper sorting
- 🗜️ Recompresses CBZ files for optimized storage
- 🚀 Processes multiple files concurrently for speed
- 🧹 Automatic cleanup of temporary files
- 📝 Automatic file renaming with consistent series naming

## 🚀 Usage

### Basic Usage

1. Place the `cbz-converter` executable in the directory containing your CBR/CBZ/PDF files. You can find it [here](https://github.com/Romaixn/cbz-converter/releases/latest).
2. Run the program:
   ```
   ./cbz-converter
   ```
3. The tool will automatically process all CBR, CBZ, and PDF files in the current directory.

### Renaming Files

You can automatically rename your comic files to follow a consistent naming pattern:

```
./cbz-converter --name "Series Name"
```

Or use positional arguments:

```
./cbz-converter "Series Name"
```

This will rename all CBR/CBZ/PDF files to the format "Series Name T01.cbz", "Series Name T02.cbz", etc., based on the tome/volume number detected in the original filename.

## 🎭 How It Works

1. 📂 Scans the current directory for CBR, CBZ, and PDF files
2. 📤 Extracts the contents of each archive or PDF
3. 🔢 Renames image files with leading zeros (e.g., 1.jpg → 001.jpg)
4. 📝 If a series name is provided, renames the files with a consistent naming pattern
5. 🔄 For CBR files: Creates a new CBZ archive and deletes the original CBR
6. 📄 For PDF files: Extracts images, creates a CBZ archive, and deletes the original PDF
7. 🗜️ For CBZ files: Recompresses the archive with the renamed files
8. 🧹 Cleans up temporary extraction directories

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the GNU General Public License v3.0 License - see the [LICENSE](LICENSE) file for details.

Happy comic reading! 📚🦸‍♂️🦸‍♀️
