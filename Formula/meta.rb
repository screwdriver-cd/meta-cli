class Meta < Formula
  @@bin_name = "meta-cli_#{Gem::Platform.local.os}_#{Hardware::CPU.arch}"
  @@shas = {
    darwin: {
      arm64: '02636a1b49373eda6ad135d8f9c25907f06611d21846a7357bbf7235b54d2b48',
      amd64: '236609f801a5990eba7b76f01d6cdc71aa6aa35dbe8fcecf79a24879c007bb72'
    },
    linux: {
      arm64: '41760396c21a371d2ee4195ca60274b644d6dec257dae4864178542d0cad01b9',
      amd64: '6f10b006386cc39d89b59057e647734ec59857f231f3369a88aa1bcf23ce28e9'
    }
  }

  desc 'CLI for reading/writing project metadata'
  homepage 'https://github.com/screwdriver-cd/meta-cli'
  version '0.0.58'
  url "https://github.com/screwdriver-cd/meta-cli/releases/download/v#{version}/#{@@bin_name}"
  sha256 @@shas[Gem::Platform.local.os.to_sym][Hardware::CPU.arch.to_sym]

  def install
    h = {}
    h[@@bin_name] = 'meta'
    bin.install h
  end
end
