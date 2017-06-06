import os
import shutil
import snapcraft

class DoctlBuildPlugin(snapcraft.BasePlugin):

    def __init__(self, name, options, project):
        super().__init__(name, options, project)

    def build(self):
        result = self.run([os.path.join(self.sourcedir, 'scripts/build.sh')])
        shutil.move(os.path.join(self.sourcedir, "out"), self.installdir)
        return result
            
