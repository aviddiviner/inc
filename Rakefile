GIT_COMMIT   = `git rev-parse --short HEAD`.chomp
GIT_TAG      = `git tag -l --points-at HEAD`.chomp  # requires git 1.7.10+
GIT_IS_DIRTY = begin `git diff-index --quiet HEAD`; $?.exitstatus == 1 end

BUILD_DATE   = Time.now.strftime('%Y-%m-%d %H:%M')
BUILD_COMMIT = "#{GIT_TAG} #{GIT_COMMIT}#{GIT_IS_DIRTY ? '*' : ''}".lstrip

VERSION_FILE = 'version.go'
VERSION_DATA = <<EOS
package main
const (
  BUILD_DATE = "#{BUILD_DATE}"
  BUILD_COMMIT = "#{BUILD_COMMIT}"
)
EOS

DEPENDENCIES = %w(
  golang.org/x/crypto/pbkdf2
  github.com/mitchellh/goamz/aws
  github.com/stretchr/testify/assert
  github.com/docopt/docopt-go
)

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
  namespace :install do
    desc 'install all dependencies'
    task :deps do
      DEPENDENCIES.each do |dep|
        sh "go get #{dep}"
      end
    end
  end
  desc 'compile packages'
  task :build => :version do
    sh 'go build'
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

desc 'watch files for changes and run tests'
task :guard do
  sh 'bundle exec guard'
end

task :init => ['go:install:deps']
task :default => ['go:version', 'go:test', 'go:build', 'go:install']
