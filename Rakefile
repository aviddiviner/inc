GIT_COMMIT   = `git rev-parse --short HEAD`.chomp
GIT_TAG      = `git tag -l --points-at HEAD`.chomp  # requires git 1.7.10+
GIT_IS_DIRTY = begin `git diff-index --quiet HEAD`; $?.exitstatus == 1 end

BUILD_DATE   = Time.now.strftime('%Y-%m-%d %H:%M')
BUILD_COMMIT = "#{GIT_TAG} #{GIT_COMMIT}#{GIT_IS_DIRTY ? '*' : ''}".lstrip

VERSION_FILE = 'version.go'
VERSION_DATA = <<EOS
package main

const (
\tBUILD_DATE   = "#{BUILD_DATE}"
\tBUILD_COMMIT = "#{BUILD_COMMIT}"
)
EOS

namespace :go do
  desc 'write the version file with build date and last commit'
  task :version do
    File.write(VERSION_FILE, VERSION_DATA)
    puts "build #{BUILD_DATE} (#{BUILD_COMMIT}) > #{VERSION_FILE}"
  end
  desc 'install packages'
  task :install => :build do
    sh 'go install'
  end
  desc 'build packages'
  task :build => :version do
    sh 'dep ensure'
    sh 'go build'
  end
  desc 'simplify code with gofmt -s'
  task :fmt do
    sh 'git ls-files | grep ".go$" | xargs gofmt -l -s -w'
  end
  desc 'run all tests, excluding long-running tests named TestX...'
  task :test do
    sh 'go test -cover -run "^Test[^X]" ./...'
  end
  namespace :test do
    desc 'run all tests and benchmarks'
    task :bench do
      sh 'go test -v -bench . ./... | grep --color -E "FAIL|$"'
    end
  end
end

task :init => ['go:install:deps']
task :default => ['go:version', 'go:test', 'go:build', 'go:install']
