# frozen_string_literal: true

require 'bundler/setup'
require 'kk/git/rake_tasks'


task default: %w[push]

desc "Build and restart"
task :run do
  sh "docker", "compose", "up", "-d", "--build", "--force-recreate"
end

task :push do
  Rake::Task['git:auto_commit_push'].invoke
end


