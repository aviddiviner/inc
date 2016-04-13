require 'rake'

def git_is_dirty
  `git diff-index --quiet HEAD`
  $?.exitstatus == 1
end

# Just having some fun :) Comparable int for `git --version` string.
def git_version_to_int(str)
  str.split('.').map(&:to_i).zip([3,2,1,0]).
    map{ |p, e| p << (e*7) }.reduce(0, :+)
end

BUILD_DATE   = Time.now.strftime('%Y-%m-%d %H:%M')

git_version  = `git --version`.split.last
git_commit   = `git rev-parse --short HEAD`.chomp

if git_version_to_int(git_version) > git_version_to_int("1.7.10")
  git_tag    = `git tag -l --points-at HEAD`.chomp  # --points-at requires git 1.7.10
else
  tags       = `git show-ref --tags`.split("\n")
  git_tag    = tags.grep(/^#{git_commit}/).join.scan(/^.* refs\/tags\/(.*)/).join
end

BUILD_COMMIT = "#{git_tag.empty? ? '' : git_tag + ' '}" +
               "#{git_commit}" +
               "#{git_is_dirty ? '*' : ''}"

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
