package modules

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"bex/utils"
)

type BackupConfig struct {
	TargetDir  string
	BackupDir  string
	BackupTime int
	Repeat     bool
	MaxBackups int
}

type BackupStats struct {
	FileCount int
	TotalSize int64
	BackupNum int
}

func WorldBackup(
	targetDir string,
	outputDir string,
	backupTime string,
	repeat bool,
	maxBackups int,
) {
	backup := &WorldBackupInstance{
		Config: &BackupConfig{},
		Stats:  &BackupStats{},
	}

	backup.ParseParams(targetDir, outputDir, backupTime, repeat, maxBackups)
	backup.Execute()
}

type WorldBackupInstance struct {
	Config    *BackupConfig
	Stats     *BackupStats
	Running   bool
	StartTime time.Time
}

func (w *WorldBackupInstance) ParseParams(
	targetDir string,
	outputDir string,
	backupTime string,
	repeat bool,
	maxBackups int,
) {
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		absTarget = targetDir
	}
	w.Config.TargetDir = absTarget
	w.Config.Repeat = repeat
	w.Config.MaxBackups = maxBackups

	if outputDir != "" {
		w.Config.BackupDir = outputDir
	} else {
		w.Config.BackupDir = filepath.Join(filepath.Dir(absTarget), "bex_backup")
	}

	if backupTime != "" {
		w.Config.BackupTime = utils.ParseTimeString(backupTime)
	}
}

func (w *WorldBackupInstance) Execute() {
	if !w.validateArgs() {
		return
	}

	utils.LogInfo("备份目标目录: %s", w.Config.TargetDir)
	utils.LogInfo("备份保存目录: %s", w.Config.BackupDir)

	if w.Config.BackupTime > 0 {
		utils.LogInfo("备份间隔: %s", w.formatDuration(w.Config.BackupTime))
		if w.Config.MaxBackups > 0 {
			utils.LogInfo("最大备份次数: %d", w.Config.MaxBackups)
		}
	}

	err := os.MkdirAll(w.Config.BackupDir, 0755)
	if err != nil {
		utils.LogError("无法创建备份目录: %s", w.Config.BackupDir)
		return
	}

	w.Running = true
	w.StartTime = time.Now()

	if w.Config.Repeat && w.Config.BackupTime > 0 {
		w.loopBackup()
	} else if w.Config.BackupTime > 0 {
		utils.LogInfo("等待 %s 后执行备份...", w.formatDuration(w.Config.BackupTime))
		w.countdown(w.Config.BackupTime)
		w.performBackup()
	} else {
		w.performBackup()
	}
}

func (w *WorldBackupInstance) validateArgs() bool {
	if !utils.FileExists(w.Config.TargetDir) {
		utils.LogError("备份目标目录不存在: %s", w.Config.TargetDir)
		return false
	}

	if w.Config.BackupTime < 0 {
		utils.LogError("备份间隔不能为负数")
		return false
	}

	if w.Config.MaxBackups < 0 {
		utils.LogError("最大备份次数必须大于等于 0")
		return false
	}

	return true
}

func (w *WorldBackupInstance) performBackup() {
	w.Stats.BackupNum++

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	dirName := filepath.Base(w.Config.TargetDir)
	backupName := fmt.Sprintf("%s_backup_%s.zip", dirName, timestamp)
	backupPath := filepath.Join(w.Config.BackupDir, backupName)

	utils.LogInfo("开始执行第 %d 次备份...", w.Stats.BackupNum)

	zipFile, err := os.Create(backupPath)
	if err != nil {
		utils.LogError("创建备份文件失败: %s", err)
		return
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {

		}
	}(zipFile)

	zipWriter := zip.NewWriter(zipFile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {

		}
	}(zipWriter)

	fileCount, err := w.addToZip(zipWriter, w.Config.TargetDir, dirName)
	if err != nil {
		utils.LogError("添加目录失败 %s: %s", dirName, err)
		return
	}

	fileInfo, err := zipFile.Stat()
	if err != nil {
		utils.LogError("获取文件信息失败: %s", err)
		return
	}

	w.Stats.FileCount = fileCount
	w.Stats.TotalSize = fileInfo.Size()

	utils.LogInfo("%s已将存档备份至 %s (%s, 共 %d 个文件)%s",
		utils.ColorGreen, backupName, utils.FormatFileSize(fileInfo.Size()), fileCount, utils.ColorClear)
}

func (w *WorldBackupInstance) addToZip(zipWriter *zip.Writer, sourceDir, basePath string) (int, error) {
	tmpDir, err := os.MkdirTemp("", "bex_backup_*")
	if err != nil {
		return 0, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	tmpCopy := filepath.Join(tmpDir, "copy")

	skipped := 0
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return nil
		}

		destPath := filepath.Join(tmpCopy, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		src, err := os.Open(path)
		if err != nil {
			utils.LogWarn("跳过被锁定文件: %s", filepath.Base(path))
			skipped++
			return nil
		}
		defer func() { _ = src.Close() }()

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil
		}

		dst, err := os.Create(destPath)
		if err != nil {
			skipped++
			return nil
		}
		defer func() { _ = dst.Close() }()

		_, _ = io.Copy(dst, src)
		return nil
	})
	if err != nil {
		return 0, err
	}

	if skipped > 0 {
		utils.LogWarn("共跳过 %d 个被锁定的文件", skipped)
	}

	fileCount := 0
	err = filepath.Walk(tmpCopy, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}

		relPath, err := filepath.Rel(tmpCopy, path)
		if err != nil {
			return nil
		}
		if relPath == "." {
			return nil
		}

		relPathSlash := filepath.ToSlash(relPath)
		baseSlash := filepath.ToSlash(basePath)
		zipPath := strings.TrimRight(baseSlash, "/") + "/" + relPathSlash

		if info.IsDir() {
			_, err = zipWriter.Create(zipPath + "/")
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = zipPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() { _ = file.Close() }()

		_, err = io.Copy(writer, file)
		if err == nil {
			fileCount++
		}
		return nil
	})

	return fileCount, err
}

func (w *WorldBackupInstance) loopBackup() {
	utils.LogInfo("使用 Ctrl+C 取消备份")

	ticker := time.NewTicker(time.Duration(w.Config.BackupTime) * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	for {
		select {
		case <-ticker.C:
			if w.Config.MaxBackups > 0 && w.Stats.BackupNum >= w.Config.MaxBackups {
				utils.LogInfo("已达到最大备份次数")
				return
			}

			if w.Stats.BackupNum > 0 {
				w.countdown(w.Config.BackupTime)
			}

			w.performBackup()

		case <-sigChan:
			utils.LogInfo("备份任务已取消")
			return
		}
	}
}

func (w *WorldBackupInstance) countdown(seconds int) {
	if seconds <= 0 {
		return
	}

	nextNum := w.Stats.BackupNum + 1
	maxStr := ""
	if w.Config.MaxBackups > 0 {
		maxStr = fmt.Sprintf("/%d", w.Config.MaxBackups)
	}

	for i := seconds; i > 0; i-- {
		timeStr := w.formatDuration(i)
		utils.LogInfo("\r正在进行第 %d%s 次备份，下次备份将在 %s后执行",
			nextNum, maxStr, timeStr)
		time.Sleep(1 * time.Second)
	}
	fmt.Println()
}

func (w *WorldBackupInstance) formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d 秒", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		secs := seconds % 60
		if secs == 0 {
			return fmt.Sprintf("%d 分钟", minutes)
		}
		return fmt.Sprintf("%d 分 %d 秒", minutes, secs)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		secs := seconds % 60
		if minutes == 0 && secs == 0 {
			return fmt.Sprintf("%d 小时", hours)
		} else if secs == 0 {
			return fmt.Sprintf("%d 小时 %d 分", hours, minutes)
		}
		return fmt.Sprintf("%d 小时 %d 分 %d 秒", hours, minutes, secs)
	}
}
