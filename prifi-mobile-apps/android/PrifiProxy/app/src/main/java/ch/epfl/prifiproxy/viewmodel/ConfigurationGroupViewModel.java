package ch.epfl.prifiproxy.viewmodel;

import android.app.Application;
import android.arch.lifecycle.AndroidViewModel;
import android.arch.lifecycle.LiveData;
import android.support.annotation.NonNull;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;

import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.repository.ConfigurationGroupRepository;

public class ConfigurationGroupViewModel extends AndroidViewModel {
    private ConfigurationGroupRepository repository;
    private LiveData<List<ConfigurationGroup>> allGroups;

    public ConfigurationGroupViewModel(@NonNull Application application) {
        super(application);
        repository = ConfigurationGroupRepository.getInstance(application);
        allGroups = repository.getAllGroups();
    }

    public LiveData<List<ConfigurationGroup>> getAllGroups() {
        return allGroups;
    }

    public void insert(ConfigurationGroup group) {
        repository.insert(group);
    }

    public void update(List<ConfigurationGroup> groups) {
        repository.update(groups);
    }

    public void delete(List<ConfigurationGroup> groups) {
        repository.delete(groups);
    }

    public void setActiveGroup(ConfigurationGroup group, boolean isActive) {
        repository.setActiveGroup(group, isActive);
    }
}
