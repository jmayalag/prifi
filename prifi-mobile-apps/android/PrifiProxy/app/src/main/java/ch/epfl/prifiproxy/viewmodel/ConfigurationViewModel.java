package ch.epfl.prifiproxy.viewmodel;

import android.app.Application;
import android.arch.lifecycle.AndroidViewModel;
import android.arch.lifecycle.LiveData;
import android.support.annotation.NonNull;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.repository.ConfigurationGroupRepository;
import ch.epfl.prifiproxy.repository.ConfigurationRepository;

public class ConfigurationViewModel extends AndroidViewModel {
    private ConfigurationRepository configurationRepository;
    private ConfigurationGroupRepository groupRepository;

    private LiveData<ConfigurationGroup> group;
    private LiveData<List<Configuration>> configurations;
    private List<Configuration> deleteList;

    public ConfigurationViewModel(@NonNull Application application) {
        super(application);
        configurationRepository = ConfigurationRepository.getInstance(application);
        groupRepository = ConfigurationGroupRepository.getInstance(application);
    }

    public void init(int groupId) {
        group = groupRepository.getGroup(groupId);
        configurations = configurationRepository.getConfigurations(groupId);
        deleteList = new ArrayList<>();
    }

    public LiveData<ConfigurationGroup> getGroup() {
        return group;
    }

    public LiveData<List<Configuration>> getConfigurations() {
        return configurations;
    }

    public void insert(Configuration configuration) {
        configurationRepository.insert(configuration);
    }

    public void update(Configuration configuration) {
        configurationRepository.update(configuration);
    }

    public void delete(Configuration configuration) {
        configurationRepository.delete(configuration);
    }

    public void insert(ConfigurationGroup group) {
        groupRepository.insert(group);
    }

    public void insertOrUpdate(ConfigurationGroup group) {
        if (group.getId() == 0) {
            insert(group);
        } else {
            update(group);
        }
    }

    public void update(ConfigurationGroup group) {
        groupRepository.update(Collections.singletonList(group));
    }

    public void delete(ConfigurationGroup group) {
        groupRepository.delete(Collections.singletonList(group));
    }

    public void toDelete(Configuration item) {
        deleteList.add(item);
    }

    /**
     * Deletes swiped items added {@link ConfigurationViewModel#toDelete(Configuration)} )}
     */
    public void performDelete() {
        configurationRepository.delete(deleteList.toArray(new Configuration[deleteList.size()]));
    }

    public void updatePriorities(List<Configuration> newOrder) {
        int priority = 1;
        for (Configuration configuration : newOrder) {
            if (deleteList.contains(configuration)) continue; // skip deleted items
            configuration.setPriority(priority);
            priority += 1;
        }

        configurationRepository.update(newOrder);
    }

    public int willDeleteCount() {
        return deleteList.size();
    }
}
