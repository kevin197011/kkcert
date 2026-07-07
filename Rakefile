# frozen_string_literal: true

require 'bundler/setup'
require 'kk/git/rake_tasks'


task default: %w[push]

COMPOSE = %w[docker compose].freeze

def compose(*args)
  sh(*COMPOSE, *args)
end

desc "Build Docker image"
task :build do
  compose "build"
end

desc "Build and start in background"
task :up do
  compose "up", "-d", "--build"
end

task run: :up

desc "Stop and remove containers"
task :down do
  compose "down"
end

task stop: :down

desc "Follow container logs"
task :logs do
  compose "logs", "-f"
end

desc "Rebuild and force-recreate containers"
task :restart do
  compose "up", "-d", "--build", "--force-recreate"
end

desc "Show container status"
task :ps do
  compose "ps"
end

desc "Run in foreground (debug)"
task :dev do
  compose "up", "--build"
end

desc "Frontend dev server with HMR (run rake up first)"
task "dev-frontend" do
  Dir.chdir("frontend") { sh "npm", "run", "dev" }
end

task :push do
  Rake::Task['git:auto_commit_push'].invoke
end


