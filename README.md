# 📚 CBZ Converter

![Go version](https://img.shields.io/github/go-mod/go-version/Romaixn/cbz-converter)
![License](https://img.shields.io/github/license/Romaixn/cbz-converter)
![GitHub Release](https://img.shields.io/github/v/release/Romaixn/cbz-converter)

## ✨ Features

- 🔄 Converts CBR files to CBZ format
- 🔢 Renames image files with leading zeros for proper sorting
- 🗜️ Recompresses CBZ files for optimized storage
- 🚀 Processes multiple files concurrently for speed
- 🧹 Automatic cleanup of temporary files
- 📝 Automatic file renaming with consistent series naming

## 🛠️ Installation

1. Ensure you have Go 1.23 or later installed on your system.
2. Clone this repository:
   ```
   git clone https://github.com/Romaixn/cbz-converter.git
   ```
3. Navigate to the project directory:
   ```
   cd cbz-converter
   ```
4. Build the project:
   ```
   go build -o cbz-converter
   ```

## 🚀 Usage

### Basic Usage

1. Place the `cbz-converter` executable in the directory containing your CBR/CBZ files. You can find it [here](https://github.com/Romaixn/cbz-converter/releases/latest).
2. Run the program:
   ```
   ./cbz-converter
   ```
3. The tool will automatically process all CBR and CBZ files in the current directory.

### Renaming Files

You can automatically rename your comic files to follow a consistent naming pattern:

```
./cbz-converter --name "Series Name"
```

Or use positional arguments:

```
./cbz-converter "Series Name"
```

This will rename all CBR/CBZ files to the format "Series Name T01.cbz", "Series Name T02.cbz", etc., based on the tome/volume number detected in the original filename.

## 🎭 How It Works

1. 📂 Scans the current directory for CBR and CBZ files
2. 📤 Extracts the contents of each archive
3. 🔢 Renames image files with leading zeros (e.g., 1.jpg → 001.jpg)
4. 📝 If a series name is provided, renames the files with a consistent naming pattern
5. 🔄 For CBR files: Creates a new CBZ archive and deletes the original CBR
6. 🗜️ For CBZ files: Recompresses the archive with the renamed files
7. 🧹 Cleans up temporary extraction directories

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the GNU General Public License v3.0 License - see the [LICENSE](LICENSE) file for details.

Happy comic reading! 📚🦸‍♂️🦸‍♀️
