# 
# cboxswanapid spec file
#

Name: cboxswanapid
Summary: A server that allows SWAN to share with users and groups.
Version: 1.1.6
Release: 1%{?dist}
License: AGPLv3
BuildRoot: %{_tmppath}/%{name}-buildroot
Group: CERN-IT/ST
BuildArch: x86_64
Source: %{name}-%{version}.tar.gz

%description
This RPM provides a golang webserver that provides a share API for SWAN

# Don't do any post-install weirdness, especially compiling .py files
%define __os_install_post %{nil}

%prep
%setup -n %{name}-%{version}

%install
# server versioning

# installation
rm -rf %buildroot/
mkdir -p %buildroot/usr/local/bin
mkdir -p %buildroot/etc/cboxswanapid
mkdir -p %buildroot/etc/logrotate.d
mkdir -p %buildroot/usr/lib/systemd/system
mkdir -p %buildroot/var/log/cboxswanapid
install -m 755 cboxswanapid	     %buildroot/usr/local/bin/cboxswanapid
install -m 644 cboxswanapid.service    %buildroot/usr/lib/systemd/system/cboxswanapid.service
install -m 644 cboxswanapid.yaml       %buildroot/etc/cboxswanapid/cboxswanapid.yaml
install -m 644 cboxswanapid.logrotate  %buildroot/etc/logrotate.d/cboxswanapid

%clean
rm -rf %buildroot/

%preun

%post

%files
%defattr(-,root,root,-)
/etc/cboxswanapid
/etc/logrotate.d/cboxswanapid
/var/log/cboxswanapid
/usr/lib/systemd/system/cboxswanapid.service
/usr/local/bin/*
%config(noreplace) /etc/cboxswanapid/cboxswanapid.yaml


%changelog
* Sat Feb 13 2021 Diogo Castro <diogo.castro@cern.ch> 1.1.6
- OIDC authentication
* Mon Dec 12 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.1.5
- Fix hardcoded configuration
* Mon Dec 11 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.1.4
- Add flag to specify script to run the share actions
* Thu Nov 30 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.1.3
- Fix deleteShare handler
* Thu Nov 29 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.1.2
- send CORS headers on GET for search endpoint
* Thu Nov 28 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.1.1
- add CORS support to search endpoint
* Thu Nov 28 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.1.0
- search endpoint now uses query params instead of url param
* Thu Nov 27 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.0.3
- Use actions without swan prefixes
* Thu Nov 27 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.0.2
- Use gorilla/context instead of native context package as it breaks path params
* Thu Nov 23 2017 Hugo Gonzalez Labrador <hugo.gonzalez.labrador@cern.ch> 1.0.0
- v1.0.0

