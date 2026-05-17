use libc::{c_char, c_int, c_uchar};
use std::ffi::{CStr, CString};
use libloading::{Library, Symbol};

pub struct SafeSocket {
    lib: &'static Library,
    handle: i32,
}

pub struct SafeSocketConnection {
    lib: &'static Library,
    handle: i32,
}

#[derive(Debug, Clone)]
pub struct SocketConfig {
    pub public_ip: String,
    pub deadline_ms: i32,
    pub heartbeat_interval_ms: i32,
    pub handshake_timeout_ms: i32,
}

impl Default for SocketConfig {
    fn default() -> Self {
        SocketConfig {
            public_ip: "".to_string(),
            deadline_ms: 0,
            heartbeat_interval_ms: 0,
            handshake_timeout_ms: 0,
        }
    }
}

impl SafeSocket {
    pub fn new(profile_name: &str, address: &str, config: Option<SocketConfig>, socket_type: &str, auto_connect: bool, lib_path: &str) -> Result<Self, Box<dyn std::error::Error>> {
        // We use a static reference and leak the library because Go's runtime 
        // does not support being unloaded (dlclose) and will hang.
        let lib = Box::leak(Box::new(unsafe { Library::new(lib_path)? }));
        let config = config.unwrap_or_default();
        
        let handle = unsafe {
            let func: Symbol<unsafe extern "C" fn(*const c_char, *const c_char, *const c_char, *const c_char, c_int, c_int, c_int, c_int) -> i32> = 
                lib.get(b"SafeSocket_CreateExtended")?;
            
            let profile_c = CString::new(profile_name)?;
            let address_c = CString::new(address)?;
            let public_ip_c = CString::new(config.public_ip)?;
            let socket_type_c = CString::new(socket_type)?;
            
            func(
                profile_c.as_ptr(),
                address_c.as_ptr(),
                public_ip_c.as_ptr(),
                socket_type_c.as_ptr(),
                config.handshake_timeout_ms,
                config.deadline_ms,
                config.heartbeat_interval_ms,
                if auto_connect { 1 } else { 0 }
            )
        };

        if handle == -1 {
            let err_func: Symbol<unsafe extern "C" fn() -> *const c_char> = unsafe { lib.get(b"SafeSocket_GetSocketError")? };
            let err_ptr = unsafe { err_func() };
            let err_msg = if err_ptr.is_null() { "Unknown error".to_string() } else { unsafe { CStr::from_ptr(err_ptr).to_string_lossy().into_owned() } };
            return Err(err_msg.into());
        }

        Ok(SafeSocket { lib, handle })
    }

    pub fn open(&self) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32) -> i32> = self.lib.get(b"SafeSocket_Open")?;
            if func(self.handle) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    pub fn close(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        if self.handle != -1 {
            unsafe {
                let func: Symbol<unsafe extern "C" fn(i32) -> i32> = self.lib.get(b"SafeSocket_Close")?;
                if func(self.handle) == -1 {
                    return Err(self.get_last_error().into());
                }
                self.handle = -1;
            }
        }
        Ok(())
    }

    pub fn send(&self, data: &[u8]) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, *const c_uchar, c_int) -> i32> = self.lib.get(b"SafeSocket_Send")?;
            if func(self.handle, data.as_ptr() as *const c_uchar, data.len() as c_int) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    pub fn receive(&self, max_length: i32) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, *mut c_uchar, c_int) -> i32> = self.lib.get(b"SafeSocket_Receive")?;
            let mut buffer = vec![0u8; max_length as usize];
            let n = func(self.handle, buffer.as_mut_ptr() as *mut c_uchar, max_length);
            if n == -1 {
                return Err(self.get_last_error().into());
            }
            buffer.truncate(n as usize);
            Ok(buffer)
        }
    }

    pub fn listen(&self) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32) -> i32> = self.lib.get(b"SafeSocket_Listen")?;
            if func(self.handle) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    pub fn accept(&self) -> Result<SafeSocketConnection, Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32) -> i32> = self.lib.get(b"SafeSocket_Accept")?;
            let conn_handle = func(self.handle);
            if conn_handle == -1 {
                return Err(self.get_last_error().into());
            }
            Ok(SafeSocketConnection { lib: self.lib, handle: conn_handle })
        }
    }

    pub fn set_deadline(&self, seconds: f64) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, f64) -> i32> = self.lib.get(b"SafeSocket_SetDeadline")?;
            if func(self.handle, seconds) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    pub fn set_idle_timeout(&self, seconds: f64) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, f64) -> i32> = self.lib.get(b"SafeSocket_SetIdleTimeout")?;
            if func(self.handle, seconds) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    fn get_last_error(&self) -> String {
        unsafe {
            let func: Symbol<unsafe extern "C" fn() -> *const c_char> = self.lib.get(b"SafeSocket_GetSocketError").unwrap();
            let ptr = func();
            if ptr.is_null() { "Unknown error".to_string() } else { CStr::from_ptr(ptr).to_string_lossy().into_owned() }
        }
    }
}

impl Drop for SafeSocket {
    fn drop(&mut self) {
        let _ = self.close();
    }
}

impl SafeSocketConnection {
    pub fn send(&self, data: &[u8]) -> Result<i32, Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, *const c_uchar, c_int) -> i32> = self.lib.get(b"SafeSocket_Send")?;
            let n = func(self.handle, data.as_ptr() as *const c_uchar, data.len() as c_int);
            if n == -1 {
                return Err(self.get_last_error().into());
            }
            Ok(n)
        }
    }

    pub fn receive(&self, max_length: i32) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, *mut c_uchar, c_int) -> i32> = self.lib.get(b"SafeSocket_Receive")?;
            let mut buffer = vec![0u8; max_length as usize];
            let n = func(self.handle, buffer.as_mut_ptr() as *mut c_uchar, max_length);
            if n == -1 {
                return Err(self.get_last_error().into());
            }
            buffer.truncate(n as usize);
            Ok(buffer)
        }
    }

    pub fn close(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        if self.handle != -1 {
            unsafe {
                let func: Symbol<unsafe extern "C" fn(i32) -> i32> = self.lib.get(b"SafeSocket_Close")?;
                if func(self.handle) == -1 {
                    return Err(self.get_last_error().into());
                }
                self.handle = -1;
            }
        }
        Ok(())
    }

    pub fn set_deadline(&self, seconds: f64) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, f64) -> i32> = self.lib.get(b"SafeSocket_SetDeadline")?;
            if func(self.handle, seconds) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    pub fn set_idle_timeout(&self, seconds: f64) -> Result<(), Box<dyn std::error::Error>> {
        unsafe {
            let func: Symbol<unsafe extern "C" fn(i32, f64) -> i32> = self.lib.get(b"SafeSocket_SetIdleTimeout")?;
            if func(self.handle, seconds) == -1 {
                return Err(self.get_last_error().into());
            }
        }
        Ok(())
    }

    fn get_last_error(&self) -> String {
        unsafe {
            let func: Symbol<unsafe extern "C" fn() -> *const c_char> = self.lib.get(b"SafeSocket_GetSocketError").unwrap();
            let ptr = func();
            if ptr.is_null() { "Unknown error".to_string() } else { CStr::from_ptr(ptr).to_string_lossy().into_owned() }
        }
    }
}

impl Drop for SafeSocketConnection {
    fn drop(&mut self) {
        let _ = self.close();
    }
}
